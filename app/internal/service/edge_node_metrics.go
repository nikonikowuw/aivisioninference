package service

import (
	"context"
	"encoding/json"
	"math"
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

// NewEdgeNodeMetricsService 创建 EdgeNodeMetricsService
func NewEdgeNodeMetricsService(metricsRepo *repository.EdgeNodeMetricsRepository, hub *ws.Hub) *EdgeNodeMetricsService {
	return &EdgeNodeMetricsService{metricsRepo: metricsRepo, hub: hub}
}

// WriteHeartbeatMetrics 将心跳中的指标写入时序表（异步非关键路径）
func (s *EdgeNodeMetricsService) WriteHeartbeatMetrics(ctx context.Context, nodeID string, req *dto.HeartbeatRequest) {
	metrics := buildEdgeNodeMetrics(nodeID, req)

	// Non-blocking goroutine with timeout to avoid slowing heartbeat response
	go func() {
		c, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := s.metricsRepo.Create(c, metrics); err != nil {
			zap.L().Error("write heartbeat metrics failed",
				zap.String("node_id", nodeID),
				zap.Error(err))
			return
		}

		// Broadcast metrics event via WebSocket (throttled per-node)
		s.broadcastMetricsEvent(nodeID, metrics)
	}()
}

func buildEdgeNodeMetrics(nodeID string, req *dto.HeartbeatRequest) *model.EdgeNodeMetrics {
	disks, err := json.Marshal(req.Disks)
	if err != nil {
		disks = []byte("[]")
	}

	memoryUsed := int64(math.Round(float64(req.HardwareInfo.TotalMemory) * req.MemoryUsage / 100))
	return &model.EdgeNodeMetrics{
		NodeID: nodeID, CreatedAt: time.Now(),
		CPUUsage: req.CPUUsage, CPULoad1m: req.CPULoad1m,
		CPULoad5m: req.CPULoad5m, CPULoad15m: req.CPULoad15m,
		MemoryUsage: req.MemoryUsage, MemoryUsed: memoryUsed,
		MemoryTotal: req.HardwareInfo.TotalMemory, DiskUsage: datatypes.JSON(disks),
		NetRXBytes: req.NetRXBytes, NetTXBytes: req.NetTXBytes,
		NetRXSpeed: req.NetRXSpeed, NetTXSpeed: req.NetTXSpeed,
		Uptime: req.Uptime, ProcessCount: req.ProcessCount,
		ThreadCount: req.ThreadCount, Temperature: req.Temperature,
		CurrentLoad: int32(req.CurrentLoad), EngineVersion: req.EngineVersion,
		HALPlatform: req.HALPlatform,
	}
}

// broadcastMetricsEvent 广播指标事件到管理端
func (s *EdgeNodeMetricsService) broadcastMetricsEvent(nodeID string, metrics *model.EdgeNodeMetrics) {
	if s.hub == nil {
		return
	}
	s.hub.Broadcast(&ws.Message{
		Type: "edge-node-metrics",
		Payload: map[string]interface{}{
			"node_id":       nodeID,
			"cpu_usage":     metrics.CPUUsage,
			"memory_usage":  metrics.MemoryUsage,
			"cpu_load_1m":   metrics.CPULoad1m,
			"cpu_load_5m":   metrics.CPULoad5m,
			"cpu_load_15m":  metrics.CPULoad15m,
			"net_rx_speed":  metrics.NetRXSpeed,
			"net_tx_speed":  metrics.NetTXSpeed,
			"process_count": metrics.ProcessCount,
			"thread_count":  metrics.ThreadCount,
			"temperature":   metrics.Temperature,
			"uptime":        metrics.Uptime,
		},
	})
}

// MetricsQueryOpts 转发 repository 的查询选项
type MetricsQueryOpts = repository.MetricsQueryOpts

// QueryMetrics 查询节点时序指标
func (s *EdgeNodeMetricsService) QueryMetrics(ctx context.Context, opts MetricsQueryOpts) ([]map[string]interface{}, int64, error) {
	return s.metricsRepo.Query(ctx, opts)
}

// DeleteOlderThan 清理过期指标记录
func (s *EdgeNodeMetricsService) DeleteOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	return s.metricsRepo.DeleteOlderThan(ctx, cutoff)
}

// GetNodeOverview 获取节点总览统计数据
func (s *EdgeNodeMetricsService) GetNodeOverview(ctx context.Context) (*dto.NodeOverviewResponse, error) {
	counts, total, err := s.metricsRepo.CountByStatus(ctx)
	if err != nil {
		return nil, err
	}

	resp := &dto.NodeOverviewResponse{
		Total: total,
	}
	for _, c := range counts {
		switch c.Status {
		case "online":
			resp.Online = c.Count
		case "offline":
			resp.Offline = c.Count
		case "error":
			resp.ErrorCount = c.Count
		case "disabled":
			resp.Disabled = c.Count
		}
	}
	return resp, nil
}
