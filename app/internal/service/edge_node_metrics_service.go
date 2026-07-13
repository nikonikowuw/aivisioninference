package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/model"
	"github.com/niko-admin/niko-admin/internal/pkg/ws"
	"github.com/niko-admin/niko-admin/internal/repository"
	"go.uber.org/zap"
	"gorm.io/datatypes"
)

// EdgeNodeMetricsService 边缘节点指标业务逻辑
type EdgeNodeMetricsService struct {
	metricsRepo *repository.EdgeNodeMetricsRepository
	hub         *ws.Hub
}

// NewEdgeNodeMetricsService creates a new EdgeNodeMetricsService.
func NewEdgeNodeMetricsService(metricsRepo *repository.EdgeNodeMetricsRepository, hub *ws.Hub) *EdgeNodeMetricsService {
	return &EdgeNodeMetricsService{
		metricsRepo: metricsRepo,
		hub:         hub,
	}
}

// BuildEdgeNodeMetrics converts a HeartbeatRequest into an EdgeNodeMetrics model for persistence.
// This is the canonical builder used by both the sync heartbeat path and the async retention path.
func BuildEdgeNodeMetrics(nodeID string, req *dto.HeartbeatRequest) *model.EdgeNodeMetrics {
	diskJSON, _ := json.Marshal(req.DiskUsage)
	if len(diskJSON) == 0 {
		diskJSON = []byte("[]")
	}

	memoryTotal := req.HardwareInfo.TotalMemory
	memoryUsed := int64(float64(memoryTotal) * req.MemoryUsage / 100.0)

	return &model.EdgeNodeMetrics{
		NodeID:    nodeID,
		CPUUsage:  req.CPUUsage,
		CPULoad1m: req.CPULoad1m,
		CPULoad5m:  req.CPULoad5m,
		CPULoad15m: req.CPULoad15m,
		MemoryUsage: req.MemoryUsage,
		MemoryUsed:  memoryUsed,
		MemoryTotal: memoryTotal,
		DiskUsage:   datatypes.JSON(diskJSON),
		NetRxBytes:  req.NetRxBytes,
		NetTxBytes:  req.NetTxBytes,
		NetRxSpeed:  req.NetRxSpeed,
		NetTxSpeed:  req.NetTxSpeed,
		Uptime:      req.Uptime,
		ProcessCount:      req.ProcessCount,
		ThreadCount:       req.ThreadCount,
		Temperature:       req.Temperature,
		WorkerCount:       req.WorkerCount,
		IdleWorkerCount:   req.IdleWorkerCount,
		ActiveStreamCount: req.ActiveStreamCount,
		DecodeSessions:    req.DecodeSessions,
		EncodeSessions:    req.EncodeSessions,
		CurrentLoad:       req.CurrentLoad,
		EngineVersion:     req.EngineVersion,
		HALPlatform:       req.HALPlatform,
	}
}

// BroadcastMetricsEvent 广播节点指标事件到管理端 WebSocket 客户端
func BroadcastMetricsEvent(hub *ws.Hub, nodeID string, metrics *model.EdgeNodeMetrics) {
	if hub == nil {
		return
	}
	hub.Broadcast(&ws.Message{
		Type: "edge-node-metrics",
		Payload: map[string]interface{}{
			"node_id":       nodeID,
			"cpu_usage":     metrics.CPUUsage,
			"memory_usage":  metrics.MemoryUsage,
			"cpu_load_1m":   metrics.CPULoad1m,
			"cpu_load_5m":   metrics.CPULoad5m,
			"cpu_load_15m":  metrics.CPULoad15m,
			"net_rx_speed":  metrics.NetRxSpeed,
			"net_tx_speed":  metrics.NetTxSpeed,
			"process_count": metrics.ProcessCount,
			"thread_count":  metrics.ThreadCount,
			"temperature":   metrics.Temperature,
			"uptime":        metrics.Uptime,
		},
	})
}

// RecordMetrics 插入一次指标记录。
func (s *EdgeNodeMetricsService) RecordMetrics(ctx context.Context, metrics *model.EdgeNodeMetrics) error {
	if metrics.NodeID == "" {
		return fmt.Errorf("node_id is required")
	}
	if metrics.CreatedAt.IsZero() {
		metrics.CreatedAt = time.Now()
	}

	if err := s.metricsRepo.Create(ctx, metrics); err != nil {
		zap.L().Error("failed to record edge node metrics",
			zap.String("node_id", metrics.NodeID),
			zap.Error(err),
		)
		return fmt.Errorf("记录指标失败: %w", err)
	}
	return nil
}

// QueryMetrics 查询指定节点的历史指标数据，支持按时间范围、指标类型、聚合方式和分页。
func (s *EdgeNodeMetricsService) QueryMetrics(ctx context.Context, nodeID string, req dto.MetricQueryRequest) (*dto.MetricQueryResponse, error) {
	items, total, err := s.metricsRepo.ListMetrics(ctx, nodeID, req)
	if err != nil {
		return nil, fmt.Errorf("查询指标失败: %w", err)
	}

	return &dto.MetricQueryResponse{
		List:  items,
		Total: total,
	}, nil
}

// DeleteOldMetrics 清理早于指定时间的指标记录。
// 由定时任务调度，默认保留 7 天数据。
func (s *EdgeNodeMetricsService) DeleteOldMetrics(ctx context.Context, retentionDays int) (int64, error) {
	cutoff := time.Now().Add(-time.Duration(retentionDays) * 24 * time.Hour)
	count, err := s.metricsRepo.DeleteOlderThan(ctx, cutoff)
	if err != nil {
		zap.L().Error("failed to delete old edge node metrics",
			zap.Time("cutoff", cutoff),
			zap.Error(err),
		)
		return 0, fmt.Errorf("清理过期指标失败: %w", err)
	}

	if count > 0 {
		zap.L().Info("deleted old edge node metrics",
			zap.Int64("count", count),
			zap.Time("cutoff", cutoff),
		)
	}

	return count, nil
}


