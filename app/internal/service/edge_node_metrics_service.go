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
	metricsRepo       *repository.EdgeNodeMetricsRepository
	engineMetricsRepo *repository.EdgeNodeEngineMetricsRepository
	hub               *ws.Hub
}

// NewEdgeNodeMetricsService creates a new EdgeNodeMetricsService.
func NewEdgeNodeMetricsService(
	metricsRepo *repository.EdgeNodeMetricsRepository,
	engineMetricsRepo *repository.EdgeNodeEngineMetricsRepository,
	hub *ws.Hub,
) *EdgeNodeMetricsService {
	return &EdgeNodeMetricsService{
		metricsRepo:       metricsRepo,
		engineMetricsRepo: engineMetricsRepo,
		hub:               hub,
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
		Type:   ws.TopicEdgeNodeMetrics,
		NodeID: nodeID,
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
	from, err := req.GetFromTime()
	if err != nil {
		return nil, fmt.Errorf("invalid from time: %w", err)
	}
	to, err := req.GetToTime()
	if err != nil {
		return nil, fmt.Errorf("invalid to time: %w", err)
	}

	// 1. Auto-select aggregation window and method to enforce 600-point limit
	duration := to.Sub(from)
	if req.Interval == "" && req.Aggregation == "" {
		req.Aggregation = "avg"
		if duration <= 1*time.Hour {
			req.Interval = "10s"
		} else if duration <= 6*time.Hour {
			req.Interval = "1m"
		} else if duration <= 24*time.Hour {
			req.Interval = "5m"
		} else if duration <= 7*24*time.Hour {
			req.Interval = "30m"
		} else {
			req.Interval = "2h"
		}
	}

	// Force page size limit for aggregation trends
	if req.PageSize == 0 || req.PageSize > 600 {
		req.PageSize = 600
	}

	// 2. Route metric query to corresponding repository
	isEngineMetric := s.engineMetricsRepo.IsEngineMetric(req.Metric)

	var items []dto.MetricDataPoint
	var total int64

	if isEngineMetric {
		if s.engineMetricsRepo == nil {
			return nil, fmt.Errorf("engine metrics repository not initialized")
		}
		items, total, err = s.engineMetricsRepo.ListMetrics(ctx, nodeID, req)
	} else {
		items, total, err = s.metricsRepo.ListMetrics(ctx, nodeID, req)
	}

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

	// Clean host metrics
	count, err := s.metricsRepo.DeleteOlderThan(ctx, cutoff)
	if err != nil {
		zap.L().Error("failed to delete old edge node host metrics",
			zap.Time("cutoff", cutoff),
			zap.Error(err),
		)
		return 0, fmt.Errorf("清理过期系统指标失败: %w", err)
	}

	// Clean engine metrics
	engineCount, err := s.engineMetricsRepo.DeleteOlderThan(ctx, cutoff)
	if err != nil {
		zap.L().Error("failed to delete old edge node engine metrics",
			zap.Time("cutoff", cutoff),
			zap.Error(err),
		)
		return count, fmt.Errorf("清理过期引擎指标失败: %w", err)
	}

	totalDeleted := count + engineCount
	if totalDeleted > 0 {
		zap.L().Info("deleted old edge node metrics physically",
			zap.Int64("host_metrics_count", count),
			zap.Int64("engine_metrics_count", engineCount),
			zap.Time("cutoff", cutoff),
		)
	}

	return totalDeleted, nil
}


