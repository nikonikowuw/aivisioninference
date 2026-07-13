package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/model"
	"github.com/niko-admin/niko-admin/internal/pkg/controlproto"
	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
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
	metricsSvc           *EdgeNodeMetricsService
	jwtManager           *jwt.Manager
	storage              storage.Storage
	minCompatibleVersion string
	versionCheckEnabled  bool
	hub                  *ws.Hub
	streamManager        *StreamManager
	nodeMuMap            sync.Map
	inferenceChan        chan *controlproto.InferenceResultParams
	stopChan             chan struct{}
	metricsRepo          *repository.EdgeNodeMetricsRepository
	alertEngine          *AlertEngine
}

// NewEdgeNodeService creates a new EdgeNodeService.
func NewEdgeNodeService(
	nodeRepo *repository.EdgeNodeRepository,
	nodeAlgoRepo *repository.EdgeNodeAlgorithmRepository,
	algoPackageRepo *repository.AlgorithmPackageRepository,
	taskRepo *repository.AIVisionTaskRepository,
	deviceRepo *repository.DeviceRepository,
	smartRecordRepo *repository.SmartRecordRepository,
	metricsSvc *EdgeNodeMetricsService,
	jwtManager *jwt.Manager,
	storage storage.Storage,
	hub *ws.Hub,
	streamManager *StreamManager,
	metricsRepo *repository.EdgeNodeMetricsRepository,
	alertEngine *AlertEngine,
) *EdgeNodeService {
	svc := &EdgeNodeService{
		nodeRepo:             nodeRepo,
		nodeAlgoRepo:         nodeAlgoRepo,
		algoPackageRepo:      algoPackageRepo,
		taskRepo:             taskRepo,
		deviceRepo:           deviceRepo,
		smartRecordRepo:      smartRecordRepo,
		metricsSvc:           metricsSvc,
		jwtManager:           jwtManager,
		storage:              storage,
		minCompatibleVersion: "",
		versionCheckEnabled:  false,
		hub:                  hub,
		streamManager:        streamManager,
		inferenceChan:        make(chan *controlproto.InferenceResultParams, 10000),
		stopChan:             make(chan struct{}),
		metricsRepo:          metricsRepo,
		alertEngine:          alertEngine,
	}
	go svc.batchInsertWorker()
	return svc
}

// SetVersionConfig sets configuration for version compatibility checking.
func (s *EdgeNodeService) SetVersionConfig(minVersion string, enabled bool) {
	s.minCompatibleVersion = minVersion
	s.versionCheckEnabled = enabled
}

// getNodeLock returns a per-node mutex to serialize concurrent heartbeat processing
// for the same node. Different nodes can still be processed in parallel.
func (s *EdgeNodeService) getNodeLock(nodeID string) sync.Locker {
	actual, _ := s.nodeMuMap.LoadOrStore(nodeID, &sync.Mutex{})
	return actual.(sync.Locker)
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
		BaseModel:              model.BaseModel{ID: nodeID},
		Name:                   req.Name,
		Description:            req.Description,
		Endpoint:               req.Endpoint,
		AuthToken:              token,
		Status:                 model.NodeStatusOffline,
		MaxLoad:                req.MaxLoad,
		MediaDecodeCapacity:    req.MediaDecodeCapacity,
		MediaEncodeCapacity:    req.MediaEncodeCapacity,
		MediaEgressCapacityBPS: req.MediaEgressCapacityBPS,
		MediaMetricsTTLSeconds: req.MediaMetricsTTLSeconds,
		Enabled:                true,
		Remark:                 req.Remark,
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
	if req.MediaDecodeCapacity != nil {
		node.MediaDecodeCapacity = *req.MediaDecodeCapacity
	}
	if req.MediaEncodeCapacity != nil {
		node.MediaEncodeCapacity = *req.MediaEncodeCapacity
	}
	if req.MediaEgressCapacityBPS != nil {
		node.MediaEgressCapacityBPS = *req.MediaEgressCapacityBPS
	}
	if req.MediaMetricsTTLSeconds != nil {
		node.MediaMetricsTTLSeconds = *req.MediaMetricsTTLSeconds
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

	return s.nodeRepo.Update(ctx, node)
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
	// Serialize heartbeats per node to prevent races on task suspension/resume.
	// Different nodes can still be processed in parallel.
	lock := s.getNodeLock(id)
	lock.Lock()
	defer lock.Unlock()

	node, err := s.findNodeByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := s.checkVersionCompatibility(id, req.EngineVersion); err != nil {
		return nil, err
	}

	// Disabled nodes: accept heartbeat for liveness tracking but skip all processing
	if node.Status == model.NodeStatusDisabled || !node.Enabled {
		now := time.Now()
		if err := s.nodeRepo.UpdateHeartbeatFields(ctx, id, map[string]interface{}{
			"last_heartbeat": &now,
		}); err != nil {
			return nil, fmt.Errorf("更新心跳字段失败: %w", err)
		}
		zap.L().Warn("heartbeat from disabled node, accepted for liveness tracking only",
			zap.String("node_id", id))
		return &dto.HeartbeatResponse{}, nil
	}

	now := time.Now()
	status := string(model.NodeStatusOnline)
	if req.Status == "error" {
		status = model.NodeStatusError
		zap.L().Info("node reported error via heartbeat",
			zap.String("node_id", id),
			zap.String("error_message", req.ErrorMessage),
		)
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
		"cpu_usage":      req.CPUUsage,
		"memory_usage":   req.MemoryUsage,
	}

	// Do NOT include "remark" in hbFields to preserve admin remark (R4).
	// Only set runtime error message to a separate field for display.
	if req.Status == "error" && req.ErrorMessage != "" {
		hbFields["runtime_error"] = req.ErrorMessage
	}

	// Update heartbeat fields with node state.
	// The per-node mutex above provides serialization for concurrent heartbeats
	// on the same node, so no DB-level transaction is needed here.
	// SyncInstalled and FindNodeOfflineSuspendedTasks use their own connections
	// to avoid SQLite "table is locked" errors that would occur if they were
	// inside a shared transaction with UpdateHeartbeatFields.
	if err := s.nodeRepo.UpdateHeartbeatFields(ctx, id, hbFields); err != nil {
		return nil, fmt.Errorf("更新心跳字段失败: %w", err)
	}

	// Sync installed algorithms
	var suspendedTasks []model.AIVisionTask
	if err := s.nodeAlgoRepo.SyncInstalled(ctx, id, req.InstalledAlgorithms); err != nil {
		return nil, fmt.Errorf("同步算法列表失败: %w", err)
	}

	// Query node-offline suspended tasks for post-tx recovery
	if status == model.NodeStatusOnline {
		tasks, err := s.taskRepo.FindNodeOfflineSuspendedTasks(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("查询暂停任务失败: %w", err)
		}
		suspendedTasks = tasks

		// Alert engine: if node was previously offline/error, resolve those alerts
		if s.alertEngine != nil && node.Status != model.NodeStatusOnline {
			if err := s.alertEngine.EvaluateNodeBackOnline(ctx, id); err != nil {
				zap.L().Error("failed to evaluate node back online alerts",
					zap.String("node_id", id),
					zap.Error(err),
				)
			}
		}
	}

	// Persist metrics snapshot to edge_node_metrics table
	if s.metricsRepo != nil {
		metrics := s.buildMetricsRecord(id, req)
		if err := s.metricsRepo.Create(ctx, metrics); err != nil {
			zap.L().Error("failed to persist edge node metrics snapshot",
				zap.String("node_id", id),
				zap.Error(err),
			)
			// Non-fatal: continue processing heartbeat even if metrics persistence fails
		}

		// Broadcast metrics event to WebSocket admin clients
		BroadcastMetricsEvent(s.hub, id, metrics)

		// Phase 2: Evaluate alert rules after metrics persistence
		if s.alertEngine != nil {
			if err := s.alertEngine.EvaluateAfterHeartbeat(ctx, id, metrics); err != nil {
				zap.L().Error("failed to evaluate alert rules",
					zap.String("node_id", id),
					zap.Error(err),
				)
				// Non-fatal: continue processing heartbeat even if alert evaluation fails
			}
		}
	}

	// Update in-memory node for downstream use (after successful transaction)
	node.LastHeartbeat = &now
	node.Uptime = req.Uptime
	node.CurrentLoad = req.CurrentLoad
	node.CPUUsage = req.CPUUsage
	node.MemoryUsage = req.MemoryUsage
	node.EngineVersion = req.EngineVersion
	node.HALPlatform = req.HALPlatform
	node.CPUModel = req.HardwareInfo.CPUModel
	node.GPUModel = req.HardwareInfo.GPUModel
	node.TotalMemory = req.HardwareInfo.TotalMemory
	node.Status = status

	// Broadcast WebSocket events after transaction commits (avoid notifying on rollback)
	if s.hub != nil {
		s.hub.Broadcast(&ws.Message{
			Type: "edge-node-status",
			Payload: map[string]interface{}{
				"node_id":        id,
				"status":         node.Status,
				"current_load":   node.CurrentLoad,
				"cpu_usage":      node.CPUUsage,
				"memory_usage":   node.MemoryUsage,
				"engine_version": node.EngineVersion,
				"last_heartbeat": node.LastHeartbeat,
			},
		})
	}

	// Attempt Engine pipeline restart for node-offline suspended tasks
	// Only restore tasks that were suspended due to node offline.
	// For each task, restart the Engine pipeline first; only mark as running
	// if Engine confirms the pipeline is active.
	if status == model.NodeStatusOnline && len(suspendedTasks) > 0 {
		s.restoreSuspendedPipelines(ctx, id, suspendedTasks)
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

// GetOverviewStats returns aggregated node status counts for the dashboard overview.
func (s *EdgeNodeService) GetOverviewStats(ctx context.Context) (*dto.OverviewStats, error) {
	counts, err := s.nodeRepo.CountByStatus(ctx)
	if err != nil {
		return nil, fmt.Errorf("get overview stats: %w", err)
	}
	var total int64
	for _, c := range counts {
		total += c
	}

	alertCount := int64(0)
	if s.alertEngine != nil {
		alertCount, err = s.alertEngine.CountActiveAlerts(ctx)
		if err != nil {
			// Non-fatal: continue with alert count = 0
			zap.L().Error("get overview stats: count active alerts failed", zap.Error(err))
		}
	}

	return &dto.OverviewStats{
		Total:      int(total),
		Online:     int(counts[model.NodeStatusOnline]),
		Offline:    int(counts[model.NodeStatusOffline]),
		Error:      int(counts[model.NodeStatusError]),
		AlertCount: int(alertCount),
	}, nil
}

// buildMetricsRecord converts a HeartbeatRequest into an EdgeNodeMetrics model for persistence.
func (s *EdgeNodeService) buildMetricsRecord(nodeID string, req *dto.HeartbeatRequest) *model.EdgeNodeMetrics {
	return BuildEdgeNodeMetrics(nodeID, req)
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

	// 删除数据库记录（设计选择："取消管理/重启后卸载"语义，见 R6）。
	// 热卸载需要 Engine MQTT 卸载命令，当前引擎尚未实现该机制。
	// 引擎重启后将不再加载该算法对应的 SO。
	// 管理端文案已同步为"移除管理"而非"卸载"以避免误导。
	return s.nodeAlgoRepo.Delete(ctx, nodeID, algoPackageID)
}

func (s *EdgeNodeService) RecommendNode(ctx context.Context, algoPackageID string) (*model.EdgeNode, error) {
	policy := NewInferenceNodePolicy(s.nodeRepo, s.nodeAlgoRepo)
	nodes, err := policy.Candidates(ctx, algoPackageID)
	if err != nil {
		return nil, fmt.Errorf("查询可用节点失败: %w", err)
	}
	if len(nodes) == 0 {
		return nil, apperrors.New(apperrors.ErrNotFound, "无可用节点")
	}

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
func (s *EdgeNodeService) PushInferenceResult(params *controlproto.InferenceResultParams) {
	select {
	case s.inferenceChan <- params:
	default:
		zap.L().Warn("EdgeNodeService: inference channel is full, dropping frame")
	}
}

// HandleLWTNodeOffline processes an MQTT Last Will and Testament (LWT) "offline" message.
// It delegates to the shared HandleNodeOffline for consistent offline processing.
func (s *EdgeNodeService) HandleLWTNodeOffline(ctx context.Context, nodeID string) error {
	lock := s.getNodeLock(nodeID)
	lock.Lock()
	defer lock.Unlock()

	node, err := s.findNodeByID(ctx, nodeID)
	if err != nil {
		return fmt.Errorf("LWT: node not found %s: %w", nodeID, err)
	}

	transitioned, err := HandleNodeOffline(ctx, s.nodeRepo, s.taskRepo, s.hub, *node,
		model.SuspendedReasonNodeOffline,
		"节点 %s 离线(MQTT LWT)，任务自动暂停", nil, node.Name)
	if err != nil {
		return fmt.Errorf("LWT: failed to process node offline %s: %w", nodeID, err)
	}
	if !transitioned {
		return nil
	}

	// Alert engine: evaluate offline rules
	if s.alertEngine != nil {
		if err := s.alertEngine.EvaluateNodeOffline(ctx, nodeID, model.NodeStatusOffline); err != nil {
			zap.L().Error("LWT: failed to evaluate offline alerts",
				zap.String("node_id", nodeID),
				zap.Error(err),
			)
		}
	}

	zap.L().Warn("LWT: edge node went offline",
		zap.String("node_id", nodeID),
		zap.String("node_name", node.Name),
	)

	return nil
}

// restoreSuspendedPipelines restarts Engine pipelines for node-offline suspended tasks.
// Only tasks with suspended_reason = 'node_offline' are processed.
// StartStream calls (the expensive part — network to Engine) run concurrently
// with a semaphore to bound concurrency. DB writes (ClearSuspended / UpdateErrorReason)
// are serialized by post-processing the results sequentially, avoiding SQLite contention
// in test environments and write-order hazards in production.
type restoreResult struct {
	task   model.AIVisionTask
	errMsg string // empty = success
}

func (s *EdgeNodeService) restoreSuspendedPipelines(ctx context.Context, nodeID string, tasks []model.AIVisionTask) {
	if len(tasks) == 0 {
		return
	}

	// Phase 1: concurrent StartStream calls (network I/O bound)
	sem := make(chan struct{}, 5)
	results := make([]restoreResult, len(tasks))

	var wg sync.WaitGroup
	wg.Add(len(tasks))

	for i, task := range tasks {
		i, task := i, task

		go func() {
			defer wg.Done()

			sem <- struct{}{}
			defer func() { <-sem }()

			result := s.restoreTaskStream(ctx, nodeID, task)
			results[i] = result
		}()
	}

	wg.Wait()

	// Phase 2: sequential DB writes (must be serialized for SQLite compatibility)
	for _, r := range results {
		if r.errMsg == "" {
			// Engine confirmed pipeline is active — clear suspension
			cleared, err := s.taskRepo.ClearNodeOfflineSuspended(ctx, r.task.ID)
			if err != nil {
				zap.L().Error("failed to clear task suspension after pipeline restore",
					zap.String("task_id", r.task.ID),
					zap.Error(err),
				)
				s.rollbackRestoredPipeline(ctx, nodeID, r.task)
				continue
			}
			if !cleared {
				zap.L().Warn("task suspension changed while pipeline was restoring",
					zap.String("task_id", r.task.ID),
					zap.String("node_id", nodeID),
				)
				s.rollbackRestoredPipeline(ctx, nodeID, r.task)
				continue
			}

			zap.L().Info("task pipeline restored after node back online",
				zap.String("task_id", r.task.ID),
				zap.String("node_id", nodeID),
			)

			if s.hub != nil {
				s.hub.Broadcast(&ws.Message{
					Type: "task-status",
					Payload: map[string]interface{}{
						"task_id":          r.task.ID,
						"status":           model.TaskStatusRunning,
						"suspended_reason": nil,
						"error_reason":     "",
						"node_id":          nodeID,
					},
				})
			}
		} else {
			if updateErr := s.taskRepo.UpdateNodeOfflineSuspendedError(ctx, r.task.ID, r.errMsg); updateErr != nil {
				zap.L().Error("failed to update task error after pipeline restore failure",
					zap.String("task_id", r.task.ID),
					zap.Error(updateErr),
				)
			}
		}
	}
}

// restoreTaskStream restores a task through StreamManager so stream routing and consumer state remain consistent.
func (s *EdgeNodeService) restoreTaskStream(ctx context.Context, nodeID string, task model.AIVisionTask) restoreResult {
	if task.DeviceChannelID == "" {
		return restoreResult{
			task:   task,
			errMsg: "task has no device_channel_id",
		}
	}

	if s.streamManager == nil {
		return restoreResult{task: task, errMsg: "stream manager is unavailable"}
	}

	algoPackage, err := s.algoPackageRepo.FindByID(ctx, task.AlgoPackageID)
	if err != nil {
		zap.L().Error("failed to resolve algorithm for pipeline restore",
			zap.String("task_id", task.ID),
			zap.String("algo_package_id", task.AlgoPackageID),
			zap.Error(err),
		)
		return restoreResult{
			task:   task,
			errMsg: fmt.Sprintf("算法包解析失败，跳过恢复: %v", err),
		}
	}
	soPath, err := resolveRuntimeSoPath(algoPackage)
	if err != nil {
		return restoreResult{task: task, errMsg: fmt.Sprintf("算法运行库解析失败，跳过恢复: %v", err)}
	}

	metadata := map[string]string{
		"task_id": task.ID, "algo_package_id": task.AlgoPackageID, "target_node_id": nodeID,
		"algo_name": algoPackage.AlgorithmName, "algo_version": algoPackage.Version,
		"so_path": soPath, "algo_params_json": string(task.AIParams),
	}
	startCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	err = s.streamManager.RestoreInferenceOnNode(startCtx,
		StreamRoute{NodeID: nodeID, DeviceID: task.DeviceChannelID}, "infer:"+task.ID, metadata)

	if err != nil {
		zap.L().Error("failed to restore task pipeline",
			zap.String("task_id", task.ID),
			zap.String("node_id", nodeID),
			zap.Error(err),
		)
		return restoreResult{
			task:   task,
			errMsg: fmt.Sprintf("Engine pipeline重启失败: %v", err),
		}
	}

	return restoreResult{task: task}
}

func (s *EdgeNodeService) rollbackRestoredPipeline(ctx context.Context, nodeID string, task model.AIVisionTask) {
	if s.streamManager == nil || task.DeviceChannelID == "" {
		return
	}

	rollbackCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	if err := s.streamManager.ReleaseOnNode(rollbackCtx,
		StreamRoute{NodeID: nodeID, DeviceID: task.DeviceChannelID}, "infer:"+task.ID); err != nil {
		zap.L().Error("failed to roll back restored task pipeline",
			zap.String("task_id", task.ID),
			zap.String("node_id", nodeID),
			zap.Error(err),
		)
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

	processOne := func(params *controlproto.InferenceResultParams) {
		task, device, algo := s.resolveInferenceMetadata(params, taskCache, deviceCache, algoCache)
		record := controlproto.InferenceResultToSmartRecord(params, task.name, device, algo)
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

func (s *EdgeNodeService) resolveInferenceMetadata(params *controlproto.InferenceResultParams, taskCache map[string]cachedTask, deviceCache map[string]string, algoCache map[string]string) (task cachedTask, deviceName string, algoVersion string) {
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
