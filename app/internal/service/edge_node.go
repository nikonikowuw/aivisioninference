package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/model"
	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
	"github.com/niko-admin/niko-admin/internal/pkg/ipc"
	"github.com/niko-admin/niko-admin/internal/pkg/jwt"
	ver "github.com/niko-admin/niko-admin/internal/pkg/version"
	"github.com/niko-admin/niko-admin/internal/pkg/ws"
	"github.com/niko-admin/niko-admin/internal/repository"
	"github.com/niko-admin/niko-admin/pkg/storage"
	"go.uber.org/zap"
)

// EdgeNodeService handles business logic for EdgeNode and EdgeNodeAlgorithm operations.
type EdgeNodeService struct {
	nodeRepo             *repository.EdgeNodeRepository
	nodeAlgoRepo         *repository.EdgeNodeAlgorithmRepository
	algoPackageRepo      *repository.AlgorithmPackageRepository
	taskRepo             *repository.AIVisionTaskRepository
	deviceRepo           *repository.DeviceRepository
	smartRecordRepo      *repository.SmartRecordRepository
	jwtManager           *jwt.Manager
	storage              storage.Storage
	minCompatibleVersion string
	versionCheckEnabled  bool
	hub                  *ws.Hub
	inferenceChan        chan *ipc.InferenceResultParams
	stopChan             chan struct{}
}

// NewEdgeNodeService creates a new EdgeNodeService.
func NewEdgeNodeService(
	nodeRepo *repository.EdgeNodeRepository,
	nodeAlgoRepo *repository.EdgeNodeAlgorithmRepository,
	algoPackageRepo *repository.AlgorithmPackageRepository,
	taskRepo *repository.AIVisionTaskRepository,
	deviceRepo *repository.DeviceRepository,
	smartRecordRepo *repository.SmartRecordRepository,
	jwtManager *jwt.Manager,
	storage storage.Storage,
	hub *ws.Hub,
) *EdgeNodeService {
	svc := &EdgeNodeService{
		nodeRepo:             nodeRepo,
		nodeAlgoRepo:         nodeAlgoRepo,
		algoPackageRepo:      algoPackageRepo,
		taskRepo:             taskRepo,
		deviceRepo:           deviceRepo,
		smartRecordRepo:      smartRecordRepo,
		jwtManager:           jwtManager,
		storage:              storage,
		minCompatibleVersion: "",
		versionCheckEnabled:  false,
		hub:                  hub,
		inferenceChan:        make(chan *ipc.InferenceResultParams, 10000),
		stopChan:             make(chan struct{}),
	}
	go svc.batchInsertWorker()
	return svc
}

// SetVersionConfig sets configuration for version compatibility checking.
func (s *EdgeNodeService) SetVersionConfig(minVersion string, enabled bool) {
	s.minCompatibleVersion = minVersion
	s.versionCheckEnabled = enabled
}

func (s *EdgeNodeService) findNodeByID(ctx context.Context, id string) (*model.EdgeNode, error) {
	node, err := s.nodeRepo.FindByID(ctx, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperrors.New(apperrors.ErrNotFound, "节点不存在")
	}
	return node, err
}

// Create generates a JWT token for the node, saves the node in the DB, and returns the node & token.
func (s *EdgeNodeService) Create(ctx context.Context, req dto.CreateEdgeNodeRequest) (*model.EdgeNode, string, error) {
	exists, err := s.nodeRepo.ExistsByName(ctx, req.Name, "")
	if err != nil {
		return nil, "", fmt.Errorf("检查节点名称唯一性失败: %w", err)
	}
	if exists {
		return nil, "", apperrors.New(apperrors.ErrEdgeNodeNameTaken, "节点名称已存在")
	}

	nodeID := uuid.New().String()
	token, err := s.jwtManager.GenerateNodeToken(nodeID)
	if err != nil {
		return nil, "", fmt.Errorf("生成节点JWT令牌失败: %w", err)
	}

	node := &model.EdgeNode{
		BaseModel:   model.BaseModel{ID: nodeID},
		Name:        req.Name,
		Description: req.Description,
		Endpoint:    req.Endpoint,
		AuthToken:   token,
		Status:      model.NodeStatusOffline,
		MaxLoad:     req.MaxLoad,
		Enabled:     true,
		Remark:      req.Remark,
	}

	if err := s.nodeRepo.Create(ctx, node); err != nil {
		return nil, "", fmt.Errorf("创建节点记录失败: %w", err)
	}

	return node, token, nil
}

func (s *EdgeNodeService) List(ctx context.Context, req dto.EdgeNodeListRequest) ([]model.EdgeNode, int64, error) {
	return s.nodeRepo.List(ctx, req)
}

func (s *EdgeNodeService) GetByID(ctx context.Context, id string) (*model.EdgeNode, error) {
	return s.findNodeByID(ctx, id)
}

func (s *EdgeNodeService) Update(ctx context.Context, id string, req dto.UpdateEdgeNodeRequest) error {
	if req.Name != "" {
		exists, err := s.nodeRepo.ExistsByName(ctx, req.Name, id)
		if err != nil {
			return fmt.Errorf("检查节点名称唯一性失败: %w", err)
		}
		if exists {
			return apperrors.New(apperrors.ErrEdgeNodeNameTaken, "节点名称已存在")
		}
	}

	node, err := s.findNodeByID(ctx, id)
	if err != nil {
		return err
	}

	if req.Name != "" {
		node.Name = req.Name
	}
	if req.Description != "" {
		node.Description = req.Description
	}
	if req.Endpoint != "" {
		node.Endpoint = req.Endpoint
	}
	if req.MaxLoad > 0 {
		node.MaxLoad = req.MaxLoad
	}
	if req.Remark != "" {
		node.Remark = req.Remark
	}
	if req.Enabled != nil {
		node.Enabled = *req.Enabled
	}
	if req.Status != "" {
		node.Status = req.Status
	}

	if err := s.nodeRepo.Update(ctx, node); err != nil {
		return err
	}

	return nil
}

func (s *EdgeNodeService) Delete(ctx context.Context, id string) error {
	_, err := s.findNodeByID(ctx, id)
	if err != nil {
		return err
	}

	activeTasks, err := s.taskRepo.FindActiveByNode(ctx, id, "")
	if err != nil {
		return err
	}
	if len(activeTasks) > 0 {
		return apperrors.New(apperrors.ErrForbidden, "节点上有运行中的任务,无法删除")
	}

	return s.nodeRepo.Delete(ctx, id)
}

func (s *EdgeNodeService) HandleHeartbeat(ctx context.Context, id string, req *dto.HeartbeatRequest) (*dto.HeartbeatResponse, error) {
	node, err := s.findNodeByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := s.checkVersionCompatibility(id, req.EngineVersion); err != nil {
		return nil, err
	}

	now := time.Now()
	status := string(model.NodeStatusOnline)
	remark := ""
	if req.Status == "error" {
		status = model.NodeStatusError
		remark = req.ErrorMessage
	} else if node.Status == model.NodeStatusDisabled {
		status = model.NodeStatusDisabled
	}

	hbFields := map[string]interface{}{
		"last_heartbeat": &now,
		"uptime":         req.Uptime,
		"current_load":   req.CurrentLoad,
		"engine_version": req.EngineVersion,
		"hal_platform":   req.HALPlatform,
		"cpu_model":      req.HardwareInfo.CPUModel,
		"gpu_model":      req.HardwareInfo.GPUModel,
		"total_memory":   req.HardwareInfo.TotalMemory,
		"status":         status,
		"remark":         remark,
	}

	// Wrap all state changes in a transaction to ensure atomicity
	var suspendedTasks []model.AIVisionTask
	err = s.nodeRepo.Transaction(ctx, func(ctx context.Context) error {
		// 1. Update heartbeat fields
		if err := s.nodeRepo.UpdateHeartbeatFields(ctx, id, hbFields); err != nil {
			return fmt.Errorf("更新心跳字段失败: %w", err)
		}

		// 2. Sync installed algorithms
		if err := s.nodeAlgoRepo.SyncInstalled(ctx, id, req.InstalledAlgorithms); err != nil {
			return fmt.Errorf("同步算法列表失败: %w", err)
		}

		// 3. Resume suspended tasks if node is back online
		if status == model.NodeStatusOnline {
			tasks, err := s.taskRepo.FindSuspendedTasksByNode(ctx, id)
			if err != nil {
				return fmt.Errorf("查询暂停任务失败: %w", err)
			}
			suspendedTasks = tasks

			for _, task := range suspendedTasks {
				if err := s.taskRepo.ClearErrorReason(ctx, task.ID); err != nil {
					return fmt.Errorf("恢复任务 %s 失败: %w", task.ID, err)
				}
			}
		}

		return nil
	})

	if err != nil {
		zap.L().Error("heartbeat transaction failed",
			zap.String("node_id", id),
			zap.Error(err),
		)
		return nil, err
	}

	// Update in-memory node for downstream use (after successful transaction)
	node.LastHeartbeat = &now
	node.Uptime = req.Uptime
	node.CurrentLoad = req.CurrentLoad
	node.EngineVersion = req.EngineVersion
	node.HALPlatform = req.HALPlatform
	node.CPUModel = req.HardwareInfo.CPUModel
	node.GPUModel = req.HardwareInfo.GPUModel
	node.TotalMemory = req.HardwareInfo.TotalMemory
	node.Status = status
	node.Remark = remark

	// Broadcast WebSocket events after transaction commits (avoid notifying on rollback)
	if s.hub != nil {
		s.hub.Broadcast(&ws.Message{
			Type: "edge-node-status",
			Payload: map[string]interface{}{
				"node_id":        id,
				"status":         node.Status,
				"current_load":   node.CurrentLoad,
				"engine_version": node.EngineVersion,
				"last_heartbeat": node.LastHeartbeat,
			},
		})

		// Broadcast task resume events
		for _, task := range suspendedTasks {
			s.hub.Broadcast(&ws.Message{
				Type: "task-status",
				Payload: map[string]interface{}{
					"task_id":      task.ID,
					"status":       model.TaskStatusRunning,
					"error_reason": "",
					"node_id":      id,
				},
			})
			zap.L().Info("task resumed after node back online",
				zap.String("task_id", task.ID),
				zap.String("node_id", id),
			)
		}
	}

	pendings, err := s.nodeAlgoRepo.ListPendingByNode(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("查询待部署算法失败: %w", err)
	}

	if len(pendings) == 0 {
		return &dto.HeartbeatResponse{PendingDeployments: []dto.PendingDeployment{}}, nil
	}

	// Batch generate presigned URLs for better performance
	var deployments []dto.PendingDeployment
	hasStatusChange := false
	for _, p := range pendings {
		downloadURL, err := s.buildPresignedURL(ctx, p.AlgoPackage.PackagePath)
		if err != nil {
			zap.L().Error("failed to build presigned url for algorithm package",
				zap.String("node_id", id),
				zap.String("algo_package_id", p.AlgoPackageID),
				zap.Error(err),
			)
			continue
		}

		if err := s.nodeAlgoRepo.UpdateStatus(ctx, id, p.AlgoPackageID, model.AlgoDeployDownloading); err != nil {
			return nil, fmt.Errorf("更新算法部署状态失败: %w", err)
		}

		extractPath := p.AlgoPackage.ExtractPath
		if extractPath == "" {
			extractPath = fmt.Sprintf("%s_%s", p.AlgoPackage.AlgorithmName, p.AlgoPackage.Version)
		}

		deployments = append(deployments, dto.PendingDeployment{
			AlgoPackageID: p.AlgoPackageID,
			DownloadURL:   downloadURL,
			MD5:           p.AlgoPackage.PackageMD5,
			ExtractPath:   extractPath,
			AlgoName:      p.AlgoPackage.AlgorithmName,
			Version:       p.AlgoPackage.Version,
		})
		hasStatusChange = true
	}

	// Broadcast again so frontend picks up the downloading/pending → downloading status change
	if hasStatusChange && s.hub != nil {
		s.hub.Broadcast(&ws.Message{
			Type: "edge-node-algo-status",
			Payload: map[string]interface{}{
				"node_id": id,
			},
		})
	}

	return &dto.HeartbeatResponse{
		PendingDeployments: deployments,
	}, nil
}

func (s *EdgeNodeService) checkVersionCompatibility(nodeID, engineVersion string) error {
	if !s.versionCheckEnabled {
		return nil
	}

	compatible, err := ver.IsCompatible(engineVersion, s.minCompatibleVersion)
	if err != nil {
		zap.L().Warn("invalid engine version format",
			zap.String("node_id", nodeID),
			zap.String("engine_version", engineVersion),
			zap.Error(err),
		)
		return apperrors.New(apperrors.CodeVersionIncompatible,
			fmt.Sprintf("引擎版本格式无效: %s", engineVersion))
	}

	if !compatible {
		zap.L().Warn("engine version too old",
			zap.String("node_id", nodeID),
			zap.String("engine_version", engineVersion),
			zap.String("min_required", s.minCompatibleVersion),
		)
		return apperrors.New(apperrors.CodeVersionIncompatible,
			fmt.Sprintf("引擎版本 %s 过低,最低要求 %s,请升级引擎",
				engineVersion, s.minCompatibleVersion))
	}

	return nil
}

func (s *EdgeNodeService) buildPresignedURL(ctx context.Context, packagePath string) (string, error) {
	if s.storage == nil {
		return "", fmt.Errorf("storage driver not initialized")
	}
	return s.storage.GetPresignedURL(ctx, strings.TrimPrefix(packagePath, "/"), time.Hour)
}

func (s *EdgeNodeService) DeployAlgorithm(ctx context.Context, nodeID string, req dto.DeployAlgorithmRequest) (*dto.DeployAlgorithmResponse, error) {
	node, err := s.findNodeByID(ctx, nodeID)
	if err != nil {
		return nil, err
	}
	if node.Status != model.NodeStatusOnline {
		return nil, apperrors.NewLocalized(apperrors.ErrBadRequest, "节点离线,无法下发算法包")
	}

	_, err = s.algoPackageRepo.FindByID(ctx, req.AlgoPackageID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperrors.New(apperrors.ErrNotFound, "算法包不存在")
	}
	if err != nil {
		return nil, err
	}

	existing, err := s.nodeAlgoRepo.FindByNodeAndAlgo(ctx, nodeID, req.AlgoPackageID)
	if err == nil {
		if existing.Status == model.AlgoDeployInstalled {
			return nil, apperrors.NewLocalized(apperrors.ErrBadRequest, "算法包已安装")
		}
		if existing.Status == model.AlgoDeployPending || existing.Status == model.AlgoDeployDownloading {
			return nil, apperrors.NewLocalized(apperrors.ErrBadRequest, "算法包正在下发中")
		}

		if err := s.nodeAlgoRepo.ResetDeployment(ctx, nodeID, req.AlgoPackageID); err != nil {
			return nil, err
		}
		return &dto.DeployAlgorithmResponse{
			DeploymentID: existing.ID,
			Message:      "算法包已加入下发队列,引擎将在下次心跳时获取",
		}, nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// Record not found - attempt to restore a soft-deleted record
	restored, restoreErr := s.nodeAlgoRepo.RestoreDeleted(ctx, nodeID, req.AlgoPackageID)
	if restoreErr == nil {
		return &dto.DeployAlgorithmResponse{
			DeploymentID: restored.ID,
			Message:      "算法包已加入下发队列,引擎将在下次心跳时获取",
		}, nil
	}

	deploymentID := uuid.New().String()
	deployment := &model.EdgeNodeAlgorithm{
		BaseModel:     model.BaseModel{ID: deploymentID},
		NodeID:        nodeID,
		AlgoPackageID: req.AlgoPackageID,
		Status:        model.AlgoDeployPending,
	}

	if err := s.nodeAlgoRepo.Create(ctx, deployment); err != nil {
		return nil, fmt.Errorf("创建部署记录失败: %w", err)
	}

	return &dto.DeployAlgorithmResponse{
		DeploymentID: deploymentID,
		Message:      "算法包已加入下发队列,引擎将在下次心跳时获取",
	}, nil
}

func (s *EdgeNodeService) ListAlgorithms(ctx context.Context, nodeID string) ([]model.EdgeNodeAlgorithm, error) {
	_, err := s.findNodeByID(ctx, nodeID)
	if err != nil {
		return nil, err
	}

	return s.nodeAlgoRepo.ListByNode(ctx, nodeID)
}

func (s *EdgeNodeService) RemoveAlgorithm(ctx context.Context, nodeID string, algoPackageID string) error {
	_, err := s.findNodeByID(ctx, nodeID)
	if err != nil {
		return err
	}

	_, err = s.nodeAlgoRepo.FindByNodeAndAlgo(ctx, nodeID, algoPackageID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return apperrors.NewLocalized(apperrors.ErrNotFound, "节点算法记录不存在")
	}
	if err != nil {
		return err
	}

	// 1. 删除数据库记录
	return s.nodeAlgoRepo.Delete(ctx, nodeID, algoPackageID)
	// TODO: 可以通过 IPC 发送卸载命令给引擎,当前引擎尚未实现热卸载机制,
	// 我们暂时只删除数据库关联关系,引擎重启后将不再加载该算法。
}

func (s *EdgeNodeService) RecommendNode(ctx context.Context, algoPackageID string) (*model.EdgeNode, error) {
	nodes, err := s.nodeRepo.FindOnlineNodesWithAlgorithm(ctx, algoPackageID)
	if err != nil {
		return nil, fmt.Errorf("查询可用节点失败: %w", err)
	}
	if len(nodes) == 0 {
		return nil, apperrors.New(apperrors.ErrNotFound, "无可用节点")
	}

	sort.Slice(nodes, func(i, j int) bool {
		return loadRate(nodes[i]) < loadRate(nodes[j])
	})

	return &nodes[0], nil
}

func loadRate(node model.EdgeNode) float64 {
	maxLoad := node.MaxLoad
	if maxLoad <= 0 {
		maxLoad = 1
	}
	return float64(node.CurrentLoad) / float64(maxLoad)
}

// PushInferenceResult pushes an inference result to the batch insert queue.
func (s *EdgeNodeService) PushInferenceResult(params *ipc.InferenceResultParams) {
	select {
	case s.inferenceChan <- params:
	default:
		zap.L().Warn("EdgeNodeService: inference channel is full, dropping frame")
	}
}

// Stop gracefully shuts down the batch insert worker, flushing remaining records.
func (s *EdgeNodeService) Stop() {
	close(s.stopChan)
	close(s.inferenceChan)
}

type cachedTask struct {
	name            string
	deviceChannelID string
	algoPackageID   string
}

func (s *EdgeNodeService) batchInsertWorker() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	var batch []model.SmartRecord
	maxBatchSize := 500

	taskCache := make(map[string]cachedTask)
	deviceCache := make(map[string]string)
	algoCache := make(map[string]string)

	flush := func() {
		if len(batch) == 0 {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := s.smartRecordRepo.CreateInBatches(ctx, batch, maxBatchSize); err != nil {
			zap.L().Error("MQTT: failed to batch insert smart records", zap.Error(err))
		}
		batch = batch[:0]
	}

	processOne := func(params *ipc.InferenceResultParams) {
		task, device, algo := s.resolveInferenceMetadata(params, taskCache, deviceCache, algoCache)
		record := ipc.InferenceResultToSmartRecord(params, task.name, device, algo)
		if record != nil {
			batch = append(batch, *record)
		}

		if len(batch) >= maxBatchSize {
			flush()
		}
	}

	for {
		select {
		case <-s.stopChan:
			// Drain remaining items from inferenceChan before final flush
			for {
				select {
				case params, ok := <-s.inferenceChan:
					if !ok {
						flush()
						return
					}
					processOne(params)
				default:
					flush()
					return
				}
			}
		case params, ok := <-s.inferenceChan:
			if !ok {
				flush()
				return
			}
			processOne(params)
		case <-ticker.C:
			flush()
		}
	}
}

func (s *EdgeNodeService) resolveInferenceMetadata(params *ipc.InferenceResultParams, taskCache map[string]cachedTask, deviceCache map[string]string, algoCache map[string]string) (task cachedTask, deviceName string, algoVersion string) {
	if params.TaskID != "" {
		if t, ok := taskCache[params.TaskID]; ok {
			task = t
		} else if tObj, err := s.taskRepo.FindByID(context.Background(), params.TaskID); err == nil && tObj != nil {
			task = cachedTask{name: tObj.Name, deviceChannelID: tObj.DeviceChannelID, algoPackageID: tObj.AlgoPackageID}
			taskCache[params.TaskID] = task
		}
	}

	devID := task.deviceChannelID
	if devID == "" {
		devID = params.DeviceID
	}
	if devID != "" {
		if d, ok := deviceCache[devID]; ok {
			deviceName = d
		} else if dObj, err := s.deviceRepo.FindByID(context.Background(), devID); err == nil && dObj != nil {
			deviceName = dObj.DeviceName
			deviceCache[devID] = deviceName
		}
	}

	if task.algoPackageID != "" {
		if v, ok := algoCache[task.algoPackageID]; ok {
			algoVersion = v
		} else if aObj, err := s.algoPackageRepo.FindByID(context.Background(), task.algoPackageID); err == nil && aObj != nil {
			algoVersion = aObj.Version
			algoCache[task.algoPackageID] = algoVersion
		}
	}
	return
}
