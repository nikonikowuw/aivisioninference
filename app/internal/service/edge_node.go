package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/model"
	cachedriver "github.com/niko-admin/niko-admin/internal/pkg/cache"
	"github.com/niko-admin/niko-admin/internal/pkg/controlproto"
	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
	"github.com/niko-admin/niko-admin/internal/pkg/jwt"
	ver "github.com/niko-admin/niko-admin/internal/pkg/version"
	"github.com/niko-admin/niko-admin/internal/pkg/ws"
	"github.com/niko-admin/niko-admin/internal/repository"
	"github.com/niko-admin/niko-admin/pkg/storage"
	"go.uber.org/zap"
)

// recordHeartbeat records a node heartbeat timestamp in the store.
// Logs a warning on failure but does not return the error — the caller
// should not fail the heartbeat request due to store unavailability.
func (s *EdgeNodeService) recordHeartbeat(ctx context.Context, nodeID string, now time.Time) {
	if err := s.heartbeats.Record(ctx, nodeID, now); err != nil {
		zap.L().Warn("failed to record heartbeat",
			zap.String("node_id", nodeID), zap.Error(err))
	}
}

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
	heartbeats           HeartbeatStore
	engineMetricsStore   *EngineMetricsStore
	heartbeatTimeout     time.Duration
	runtimeStateStore    *EdgeNodeRuntimeStateStore
}

// EdgeNodeServiceConfig holds all dependencies for EdgeNodeService.
type EdgeNodeServiceConfig struct {
	NodeRepo           *repository.EdgeNodeRepository
	NodeAlgoRepo       *repository.EdgeNodeAlgorithmRepository
	AlgoPackageRepo    *repository.AlgorithmPackageRepository
	TaskRepo           *repository.AIVisionTaskRepository
	DeviceRepo         *repository.DeviceRepository
	SmartRecordRepo    *repository.SmartRecordRepository
	MetricsSvc         *EdgeNodeMetricsService
	JWTManager         *jwt.Manager
	Storage            storage.Storage
	Hub                *ws.Hub
	StreamManager      *StreamManager
	MetricsRepo        *repository.EdgeNodeMetricsRepository
	AlertEngine        *AlertEngine
	Heartbeats         HeartbeatStore
	EngineMetricsStore *EngineMetricsStore
	HeartbeatTimeout   time.Duration
	RuntimeStateStore  *EdgeNodeRuntimeStateStore
}

// NewEdgeNodeService creates a new EdgeNodeService from the given config.
func NewEdgeNodeService(cfg *EdgeNodeServiceConfig) *EdgeNodeService {
	runtimeStateStore := cfg.RuntimeStateStore
	if runtimeStateStore == nil {
		runtimeStateStore = NewEdgeNodeRuntimeStateStore(cachedriver.NewMemoryCache(0))
	}
	svc := &EdgeNodeService{
		nodeRepo:             cfg.NodeRepo,
		nodeAlgoRepo:         cfg.NodeAlgoRepo,
		algoPackageRepo:      cfg.AlgoPackageRepo,
		taskRepo:             cfg.TaskRepo,
		deviceRepo:           cfg.DeviceRepo,
		smartRecordRepo:      cfg.SmartRecordRepo,
		metricsSvc:           cfg.MetricsSvc,
		jwtManager:           cfg.JWTManager,
		storage:              cfg.Storage,
		minCompatibleVersion: "",
		heartbeats:           cfg.Heartbeats,
		versionCheckEnabled:  false,
		hub:                  cfg.Hub,
		streamManager:        cfg.StreamManager,
		inferenceChan:        make(chan *controlproto.InferenceResultParams, 10000),
		stopChan:             make(chan struct{}),
		metricsRepo:          cfg.MetricsRepo,
		alertEngine:          cfg.AlertEngine,
		engineMetricsStore:   cfg.EngineMetricsStore,
		heartbeatTimeout:     cfg.HeartbeatTimeout,
		runtimeStateStore:    runtimeStateStore,
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

// getRuntimeState 返回节点运行时状态，未命中时从 DB 回填。
func (s *EdgeNodeService) getRuntimeState(ctx context.Context, id string) (*EdgeNodeRuntimeState, error) {
	if cached, ok := s.runtimeStateStore.Get(ctx, id); ok {
		return cached, nil
	}
	node, err := s.findNodeByID(ctx, id)
	if err != nil {
		return nil, err
	}
	s.runtimeStateStore.LoadFromNode(ctx, node)
	loaded, _ := s.runtimeStateStore.Get(ctx, id)
	return loaded, nil
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
	node, err := s.findNodeByID(ctx, id)
	if err != nil {
		return nil, err
	}
	// 从内存缓存中补齐最新运行时字段，避免 edge_nodes 中的值陈旧
	s.runtimeStateStore.ApplyToNode(ctx, node)
	return node, nil
}

// RuntimeStateStore returns the node runtime-state store shared with background workers.
func (s *EdgeNodeService) RuntimeStateStore() *EdgeNodeRuntimeStateStore {
	return s.runtimeStateStore
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

	if err := s.nodeRepo.Update(ctx, node); err != nil {
		return err
	}

	// 配置变更后使缓存失效，下次心跳重新加载
	s.runtimeStateStore.Delete(ctx, id)
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

	if err := s.nodeRepo.Delete(ctx, id); err != nil {
		return err
	}

	// 删除节点后清理缓存
	s.runtimeStateStore.Delete(ctx, id)
	return nil
}

func (s *EdgeNodeService) HandleHeartbeat(ctx context.Context, id string, req *dto.HeartbeatRequest) (*dto.HeartbeatResponse, error) {
	// Serialize heartbeats per node to prevent races on task suspension/resume.
	// Different nodes can still be processed in parallel.
	lock := s.getNodeLock(id)
	lock.Lock()
	defer lock.Unlock()

	// 从内存缓存读取节点运行时状态（避免每次心跳 SELECT edge_nodes）
	runtimeState, err := s.getRuntimeState(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := s.checkVersionCompatibility(id, req.EngineVersion); err != nil {
		return nil, err
	}

	// Disabled nodes: accept heartbeat for liveness tracking but skip all processing
	if runtimeState.Status == model.NodeStatusDisabled || !runtimeState.Enabled {
		now := time.Now()
		if err := s.nodeRepo.UpdateHeartbeatFields(ctx, id, map[string]interface{}{
			"last_heartbeat": &now,
		}); err != nil {
			return nil, fmt.Errorf("更新心跳字段失败: %w", err)
		}
		s.recordHeartbeat(ctx, id, now)
		s.runtimeStateStore.UpdateFromHeartbeat(ctx, id, req, runtimeState.Status, now, runtimeState.RuntimeError)
		zap.L().Warn("heartbeat from disabled node, accepted for liveness tracking only",
			zap.String("node_id", id))
		return &dto.HeartbeatResponse{}, nil
	}

	now := time.Now()
	newStatus := model.NodeStatusOnline
	runtimeError := ""
	if req.Status == model.NodeStatusError {
		newStatus = model.NodeStatusError
		runtimeError = req.ErrorMessage
		zap.L().Info("node reported error via heartbeat",
			zap.String("node_id", id),
			zap.String("error_message", req.ErrorMessage),
		)
	}

	wasOffline := runtimeState.Status != model.NodeStatusOnline
	hbFields := map[string]interface{}{
		"last_heartbeat": &now,
		"uptime":         req.Uptime,
		"current_load":   req.CurrentLoad,
		"engine_version": req.EngineVersion,
		"hal_platform":   req.HALPlatform,
		"cpu_model":      req.HardwareInfo.CPUModel,
		"gpu_model":      req.HardwareInfo.GPUModel,
		"total_memory":   req.HardwareInfo.TotalMemory,
		"status":         newStatus,
		"cpu_usage":      req.CPUUsage,
		"memory_usage":   req.MemoryUsage,
		"runtime_error":  runtimeError,
	}
	if err := s.nodeRepo.UpdateHeartbeatFields(ctx, id, hbFields); err != nil {
		return nil, fmt.Errorf("更新心跳字段失败: %w", err)
	}

	// Publish liveness only after persistence so timeout scans can verify freshness in DB.
	s.recordHeartbeat(ctx, id, now)
	// Publish the cache after the durable update, before independent downstream work.
	s.runtimeStateStore.UpdateFromHeartbeat(ctx, id, req, newStatus, now, runtimeError)

	// Sync installed algorithms
	if err := s.nodeAlgoRepo.SyncInstalled(ctx, id, req.InstalledAlgorithms); err != nil {
		return nil, fmt.Errorf("同步算法列表失败: %w", err)
	}

	// Query node-offline suspended tasks for recovery
	var suspendedTasks []model.AIVisionTask
	if newStatus == model.NodeStatusOnline {
		tasks, err := s.taskRepo.FindNodeOfflineSuspendedTasks(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("查询暂停任务失败: %w", err)
		}
		suspendedTasks = tasks

		// 节点刚从离线/错误恢复，触发告警恢复
		if wasOffline && s.alertEngine != nil {
			if err := s.alertEngine.EvaluateNodeBackOnline(ctx, id); err != nil {
				zap.L().Error("failed to evaluate node back online alerts",
					zap.String("node_id", id), zap.Error(err))
			}
		}
	}

	// Persist metrics snapshot to edge_node_metrics table（时序数据，仍需写入）
	if s.metricsRepo != nil {
		metrics := s.buildMetricsRecord(id, req)
		if err := s.metricsRepo.Create(ctx, metrics); err != nil {
			zap.L().Error("failed to persist edge node metrics snapshot",
				zap.String("node_id", id), zap.Error(err))
		}

		BroadcastMetricsEvent(s.hub, id, metrics)

		// Phase 2: Evaluate alert rules after metrics persistence
		if s.alertEngine != nil {
			// 传入 cachedState.Name 避免 alertEngine 内部重复查询 edge_nodes
			if err := s.alertEngine.EvaluateAfterHeartbeat(ctx, id, runtimeState.Name, metrics); err != nil {
				zap.L().Error("failed to evaluate alert rules",
					zap.String("node_id", id), zap.Error(err))
			}
		}
	}

	// Broadcast WebSocket events
	if s.hub != nil {
		s.hub.Broadcast(&ws.Message{
			Type: "edge-node-status",
			Payload: map[string]interface{}{
				"node_id":        id,
				"status":         newStatus,
				"current_load":   req.CurrentLoad,
				"cpu_usage":      req.CPUUsage,
				"memory_usage":   req.MemoryUsage,
				"engine_version": req.EngineVersion,
				"last_heartbeat": &now,
			},
		})
	}

	// Restore suspended pipelines if back online
	if newStatus == model.NodeStatusOnline && len(suspendedTasks) > 0 {
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
	if _, err := s.findNodeByID(ctx, nodeID); err != nil {
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

	// 用缓存的节点状态，避免 SELECT edge_nodes
	runtimeState, err := s.getRuntimeState(ctx, nodeID)
	if err != nil {
		return fmt.Errorf("LWT: node not found %s: %w", nodeID, err)
	}

	// 构造最小化 EdgeNode 用于 HandleNodeOffline（仅需 ID 和 Name）
	node := model.EdgeNode{
		BaseModel: model.BaseModel{ID: nodeID},
		Name:      runtimeState.Name,
	}

	transitioned, err := HandleNodeOffline(ctx, s.nodeRepo, s.taskRepo, s.hub, s.heartbeats, node,
		model.SuspendedReasonNodeOffline,
		"节点 %s 离线(MQTT LWT)，任务自动暂停", nil, node.Name)
	if err != nil {
		return fmt.Errorf("LWT: failed to process node offline %s: %w", nodeID, err)
	}
	if !transitioned {
		return nil
	}

	// 成功转换后使缓存失效，下次心跳重新从 DB 加载
	s.runtimeStateStore.Delete(ctx, nodeID)

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

// AssembleCardSnapshots constructs a list of EdgeNodeCardSnapshots for a list of EdgeNodes.
func (s *EdgeNodeService) AssembleCardSnapshots(ctx context.Context, nodes []model.EdgeNode) ([]dto.EdgeNodeCardSnapshot, error) {
	snapshots := make([]dto.EdgeNodeCardSnapshot, 0, len(nodes))
	if len(nodes) == 0 {
		return snapshots, nil
	}

	// Batch query latest host metrics to eliminate N+1 database queries
	var hostMetricsMap map[string]*model.EdgeNodeMetrics
	if s.metricsRepo != nil {
		nodeIDs := make([]string, len(nodes))
		for i, n := range nodes {
			nodeIDs[i] = n.ID
		}
		var err error
		hostMetricsMap, err = s.metricsRepo.GetLatestHostMetricsByNodes(ctx, nodeIDs)
		if err != nil {
			zap.L().Warn("failed to batch query latest host metrics for nodes", zap.Error(err))
		}
	}

	// Batch query heartbeat liveness to eliminate N+1 Redis ZSCORE rounds
	var onlineMap map[string]bool
	if s.heartbeats != nil {
		nodeIDs := make([]string, len(nodes))
		for i, n := range nodes {
			nodeIDs[i] = n.ID
		}
		onlineMap, _ = s.heartbeats.BatchIsOnline(ctx, nodeIDs, s.heartbeatTimeout)
	}

	for _, node := range nodes {
		// 1. Get status from Redis HeartbeatStore (batch result map lookup)
		isOnline := false
		if onlineMap != nil {
			isOnline = onlineMap[node.ID]
		}
		status := node.Status
		if status != string(model.NodeStatusDisabled) {
			if isOnline {
				status = string(model.NodeStatusOnline)
			} else {
				status = string(model.NodeStatusOffline)
			}
		}

		// 2. Query latest host metrics from batch result
		var hostMetrics *model.EdgeNodeMetrics
		if hostMetricsMap != nil {
			hostMetrics = hostMetricsMap[node.ID]
		}

		// 3. Query latest engine metrics from EngineMetricsStore
		var engineMetrics *controlproto.EngineMetricsSnapshot
		var engineRecTime time.Time
		if s.engineMetricsStore != nil {
			engineMetrics, engineRecTime = s.engineMetricsStore.GetNodeLatest(node.ID)
		}

		// Assemble snapshot DTO
		snap := dto.EdgeNodeCardSnapshot{
			ID:          node.ID,
			Name:        node.Name,
			Description: node.Description,
			Endpoint:    node.Endpoint,
			Status:      status,
			Enabled:     node.Enabled,
			MaxLoad:     node.MaxLoad,
		}

		// Map hardware info
		snap.HardwareInfo = dto.HardwareInfo{
			CPUModel:    node.CPUModel,
			GPUModel:    node.GPUModel,
			TotalMemory: node.TotalMemory,
			CPUCores:    0,
		}

		// Map host metrics (if available)
		if hostMetrics != nil {
			snap.CPUUsage = &hostMetrics.CPUUsage
			snap.MemoryUsage = &hostMetrics.MemoryUsage
			snap.MemoryUsed = &hostMetrics.MemoryUsed
			snap.MemoryTotal = &hostMetrics.MemoryTotal
			snap.NetRxSpeed = &hostMetrics.NetRxSpeed
			snap.NetTxSpeed = &hostMetrics.NetTxSpeed
			snap.Temperature = &hostMetrics.Temperature
			snap.Uptime = &hostMetrics.Uptime
			snap.CurrentLoad = hostMetrics.CurrentLoad

			// Parse DiskUsage jsonb array
			if len(hostMetrics.DiskUsage) > 0 {
				var diskUsage []dto.DiskUsageInfo
				if err := json.Unmarshal(hostMetrics.DiskUsage, &diskUsage); err == nil {
					snap.DiskUsage = diskUsage
				}
			}
		}

		// Map engine metrics (if available)
		if engineMetrics != nil {
			snap.ActiveStreamCount = &engineMetrics.ActiveStreamCount
			snap.WorkerCount = &engineMetrics.WorkerCount
			snap.IdleWorkerCount = &engineMetrics.IdleWorkerCount
			snap.DecodeSessions = &engineMetrics.DecodeSessions
			snap.EncodeSessions = &engineMetrics.EncodeSessions
			snap.DecodeSlotsUsed = &engineMetrics.DecodeSlotsUsed
			snap.EncodeSlotsUsed = &engineMetrics.EncodeSlotsUsed
			snap.EgressBPS = &engineMetrics.EgressBPS
			snap.PreviewPipelineCount = &engineMetrics.PreviewPipelineCount
			snap.InferencePipelineCount = &engineMetrics.InferencePipelineCount
			snap.MixedPipelineCount = &engineMetrics.MixedPipelineCount
			snap.PreviewCapacity = &engineMetrics.PreviewCapacity
			snap.PreviewInUse = &engineMetrics.PreviewInUse

			if engineMetrics.AcceleratorMetricsValid {
				snap.AcceleratorUtilization = &engineMetrics.AcceleratorUtilization
				snap.AcceleratorMetricsValid = true
			}
			if !engineRecTime.IsZero() {
				recStr := engineRecTime.Format(time.RFC3339)
				snap.MetricsReceivedAt = &recStr
			}
		}

		snapshots = append(snapshots, snap)
	}

	return snapshots, nil
}
