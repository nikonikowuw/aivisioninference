package service

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"

	"github.com/niko-admin/niko-admin/internal/model"
	"github.com/niko-admin/niko-admin/internal/pkg/controlproto"
	"github.com/niko-admin/niko-admin/internal/pkg/ws"
	"github.com/niko-admin/niko-admin/internal/repository"
)

// EngineMetricsStore 引擎指标存储
// 接收 C++ 主动上报的 EngineMetricsMsg，缓存最新全局及每流指标，
// 写入 HistoryBuffer 用于趋势图，保持引擎运行状态的实时视图。
type EngineMetricsStore struct {
	mu sync.RWMutex

	// 最新全局指标
	lastMetrics *controlproto.EngineMetricsSnapshot

	// 每流最新指标 (task_id -> StreamMetricsSnapshot)
	streamMetrics map[string]*controlproto.StreamMetricsSnapshot

	// 历史缓冲区 (用于趋势图)
	historyBuffer *HistoryBuffer

	// 指标新鲜度时间戳 (最后一次收到指标的时间)
	lastUpdateTime atomic.Value // time.Time

	// 累计更新次数
	updateCount atomic.Uint64

	logger *zap.Logger
	nodes  map[string]*NodeEngineMetrics

	hub    *ws.Hub
	dbRepo *repository.EdgeNodeEngineMetricsRepository
	dbChan chan *model.EdgeNodeEngineMetrics
}

// NodeEngineMetrics is the latest metrics snapshot and receive time for one edge node.
type NodeEngineMetrics struct {
	Snapshot   *controlproto.EngineMetricsSnapshot
	ReceivedAt time.Time
}

// NewEngineMetricsStore 创建引擎指标存储
func NewEngineMetricsStore(historyBuffer *HistoryBuffer, hub *ws.Hub, dbRepo *repository.EdgeNodeEngineMetricsRepository) *EngineMetricsStore {
	store := &EngineMetricsStore{
		streamMetrics: make(map[string]*controlproto.StreamMetricsSnapshot),
		historyBuffer: historyBuffer,
		logger:        zap.L().With(zap.String("component", "engine_metrics_store")),
		nodes:         make(map[string]*NodeEngineMetrics),
		hub:           hub,
		dbRepo:        dbRepo,
		dbChan:        make(chan *model.EdgeNodeEngineMetrics, 1000),
	}
	store.lastUpdateTime.Store(time.Time{})

	go store.dbWriteWorker()

	return store
}

// Stop closes dbChan and allows dbWriteWorker to terminate gracefully.
func (s *EngineMetricsStore) Stop() {
	if s.dbChan != nil {
		close(s.dbChan)
	}
}

func (s *EngineMetricsStore) dbWriteWorker() {
	for dbModel := range s.dbChan {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := s.dbRepo.Create(ctx, dbModel); err != nil {
			s.logger.Error("Failed to persist engine metrics snapshot to database", zap.Error(err))
		}
		cancel()
	}
}

// UpdateNode updates a node-scoped snapshot without mixing metrics from other nodes.
func (s *EngineMetricsStore) UpdateNode(nodeID string, snapshot *controlproto.EngineMetricsSnapshot) {
	if nodeID == "" || snapshot == nil {
		return
	}
	now := time.Now()
	s.mu.Lock()
	s.nodes[nodeID] = &NodeEngineMetrics{Snapshot: cloneEngineMetrics(snapshot), ReceivedAt: now}
	s.mu.Unlock()
	s.Update(cloneEngineMetrics(snapshot))

	// 1. Asynchronously persist to PostgreSQL
	if s.dbRepo != nil {
		dbModel := &model.EdgeNodeEngineMetrics{
			NodeID:                 nodeID,
			CreatedAt:              now,
			ActiveStreamCount:      snapshot.ActiveStreamCount,
			DMAUsedBytes:           snapshot.DMAUsedBytes,
			DMATotalBytes:          snapshot.DMATotalBytes,
			NPUUsedBytes:           snapshot.NPUUsedBytes,
			NPUTotalBytes:          snapshot.NPUTotalBytes,
			WorkerCount:            snapshot.WorkerCount,
			IdleWorkerCount:        snapshot.IdleWorkerCount,
			DecodeSessions:         snapshot.DecodeSessions,
			EncodeSessions:         snapshot.EncodeSessions,
			DecodeSlotsUsed:        snapshot.DecodeSlotsUsed,
			EncodeSlotsUsed:        snapshot.EncodeSlotsUsed,
			EgressBPS:              snapshot.EgressBPS,
			PreviewPipelineCount:   snapshot.PreviewPipelineCount,
			InferencePipelineCount: snapshot.InferencePipelineCount,
			MixedPipelineCount:     snapshot.MixedPipelineCount,
			MediaMetricsValid:      snapshot.MediaMetricsValid,
			PreviewCapacity:        snapshot.PreviewCapacity,
			PreviewInUse:           snapshot.PreviewInUse,
			PreviewCapacityValid:   snapshot.PreviewCapacityValid,
			AcceleratorUtilization:  snapshot.AcceleratorUtilization,
			AcceleratorMetricsValid: snapshot.AcceleratorMetricsValid,
		}
		select {
		case s.dbChan <- dbModel:
		default:
			s.logger.Warn("Engine metrics DB queue is full, dropping database persistence")
		}
	}

	// 2. Broadcast to WebSocket clients
	if s.hub != nil {
		s.hub.Broadcast(&ws.Message{
			Type:    ws.TopicEdgeNodeEngineMetrics,
			NodeID:  nodeID,
			Payload: snapshot,
		})
	}
}

// GetNodeLatest returns a copy of one node's latest snapshot and its control-plane receive time.
func (s *EngineMetricsStore) GetNodeLatest(nodeID string) (*controlproto.EngineMetricsSnapshot, time.Time) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entry := s.nodes[nodeID]
	if entry == nil || entry.Snapshot == nil {
		return nil, time.Time{}
	}
	return cloneEngineMetrics(entry.Snapshot), entry.ReceivedAt
}

// GetFreshNodeMediaMetrics returns a schedulable snapshot only when both the
// engine collection timestamp and control-plane receive time are within TTL.
func (s *EngineMetricsStore) GetFreshNodeMediaMetrics(nodeID string, now time.Time, ttl time.Duration) (*controlproto.EngineMetricsSnapshot, bool) {
	snapshot, receivedAt := s.GetNodeLatest(nodeID)
	if snapshot == nil || !snapshot.PreviewCapacityValid || snapshot.PreviewCapacity == 0 ||
		snapshot.PreviewInUse > snapshot.PreviewCapacity || ttl <= 0 || receivedAt.IsZero() {
		return nil, false
	}
	collectedAt := time.Unix(0, int64(snapshot.TimestampNS))
	if snapshot.TimestampNS == 0 || collectedAt.After(now) || receivedAt.After(now) {
		return nil, false
	}
	if now.Sub(collectedAt) > ttl || now.Sub(receivedAt) > ttl {
		return nil, false
	}
	return snapshot, true
}

func cloneEngineMetrics(snapshot *controlproto.EngineMetricsSnapshot) *controlproto.EngineMetricsSnapshot {
	if snapshot == nil {
		return nil
	}
	cp := *snapshot
	if len(snapshot.Streams) > 0 {
		cp.Streams = append([]controlproto.StreamMetricsSnapshot(nil), snapshot.Streams...)
	}
	return &cp
}

// Update 更新指标 (由 MetricsReceiver 调用)
func (s *EngineMetricsStore) Update(snapshot *controlproto.EngineMetricsSnapshot) {
	if snapshot == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.lastMetrics = snapshot
	s.lastUpdateTime.Store(time.Now())
	s.updateCount.Add(1)

	// 更新每流指标
	s.streamMetrics = make(map[string]*controlproto.StreamMetricsSnapshot,
		len(snapshot.Streams))
	for i := range snapshot.Streams {
		stream := &snapshot.Streams[i]
		s.streamMetrics[stream.TaskID] = stream
	}

	// 写入历史缓冲区
	ts := time.Now()
	s.historyBuffer.Add("engine.active_streams",
		float64(snapshot.ActiveStreamCount), ts)
	s.historyBuffer.Add("engine.dma_used_mb",
		float64(snapshot.DMAUsedBytes)/1024/1024, ts)
	s.historyBuffer.Add("engine.npu_used_mb",
		float64(snapshot.NPUUsedBytes)/1024/1024, ts)
	s.historyBuffer.Add("engine.idle_workers",
		float64(snapshot.IdleWorkerCount), ts)
}

// GetLatest 获取最新全局指标
func (s *EngineMetricsStore) GetLatest() *controlproto.EngineMetricsSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.lastMetrics == nil {
		return nil
	}

	// 返回深度拷贝
	return cloneEngineMetrics(s.lastMetrics)
}

// GetStreamMetrics 获取指定流的指标
func (s *EngineMetricsStore) GetStreamMetrics(taskID string) *controlproto.StreamMetricsSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stream, ok := s.streamMetrics[taskID]
	if !ok {
		return nil
	}

	cp := *stream
	return &cp
}

// GetAllStreamMetrics 获取所有流的指标
func (s *EngineMetricsStore) GetAllStreamMetrics() []controlproto.StreamMetricsSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]controlproto.StreamMetricsSnapshot, 0, len(s.streamMetrics))
	for _, stream := range s.streamMetrics {
		result = append(result, *stream)
	}
	return result
}

// GetLastUpdateTime 获取最后更新时间
func (s *EngineMetricsStore) GetLastUpdateTime() time.Time {
	return s.lastUpdateTime.Load().(time.Time)
}

// IsStale 检查指标是否过期（超过 15 秒未更新）
func (s *EngineMetricsStore) IsStale() bool {
	return time.Since(s.GetLastUpdateTime()) > 15*time.Second
}

// GetUpdateCount 获取更新次数
func (s *EngineMetricsStore) GetUpdateCount() uint64 {
	return s.updateCount.Load()
}

// EngineMetricsSummary 引擎指标摘要（用于 API 响应）
type EngineMetricsSummary struct {
	ActiveStreams uint32  `json:"active_streams"`
	DMAUsedMB     float64 `json:"dma_used_mb"`
	DMATotalMB    float64 `json:"dma_total_mb"`
	NPUUsedMB     float64 `json:"npu_used_mb"`
	NPUTotalMB    float64 `json:"npu_total_mb"`
	WorkerCount   uint32  `json:"worker_count"`
	IdleWorkers   uint32  `json:"idle_workers"`
	LastUpdate    string  `json:"last_update"`
	UpdateCount   uint64  `json:"update_count"`
	IsStale       bool    `json:"is_stale"`
}

// GetSummary 获取摘要
func (s *EngineMetricsStore) GetSummary() *EngineMetricsSummary {
	latest := s.GetLatest()
	if latest == nil {
		return nil
	}

	return &EngineMetricsSummary{
		ActiveStreams: latest.ActiveStreamCount,
		DMAUsedMB:     float64(latest.DMAUsedBytes) / 1024 / 1024,
		DMATotalMB:    float64(latest.DMATotalBytes) / 1024 / 1024,
		NPUUsedMB:     float64(latest.NPUUsedBytes) / 1024 / 1024,
		NPUTotalMB:    float64(latest.NPUTotalBytes) / 1024 / 1024,
		WorkerCount:   latest.WorkerCount,
		IdleWorkers:   latest.IdleWorkerCount,
		LastUpdate:    s.GetLastUpdateTime().Format(time.RFC3339),
		UpdateCount:   s.GetUpdateCount(),
		IsStale:       s.IsStale(),
	}
}
