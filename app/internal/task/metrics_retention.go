package task

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"

	"github.com/niko-admin/niko-admin/internal/service"
)

// MetricsRetentionTaskType 指标数据保留清理任务类型
const MetricsRetentionTaskType = "metrics:retention"

// MetricsRetentionPayload 指标清理任务载荷
type MetricsRetentionPayload struct {
	RetentionDays int `json:"retention_days"`
}

// MetricsRetentionHandler 处理指标数据保留清理任务
type MetricsRetentionHandler struct {
	metricsSvc *service.EdgeNodeMetricsService
}

// NewMetricsRetentionHandler 创建指标清理处理器
func NewMetricsRetentionHandler(metricsSvc *service.EdgeNodeMetricsService) *MetricsRetentionHandler {
	return &MetricsRetentionHandler{
		metricsSvc: metricsSvc,
	}
}

// RegisterHandlers 注册指标清理任务处理函数
func (h *MetricsRetentionHandler) RegisterHandlers(mux *asynq.ServeMux) {
	mux.HandleFunc(MetricsRetentionTaskType, h.handleMetricsRetention)
}

// handleMetricsRetention 处理指标数据清理任务
func (h *MetricsRetentionHandler) handleMetricsRetention(ctx context.Context, t *asynq.Task) error {
	var payload MetricsRetentionPayload
	if len(t.Payload()) > 0 {
		if err := json.Unmarshal(t.Payload(), &payload); err != nil {
			return fmt.Errorf("unmarshal metrics retention payload: %w", err)
		}
	}

	retentionDays := payload.RetentionDays
	if retentionDays <= 0 {
		retentionDays = 7 // 默认保留 7 天
	}

	zap.L().Info("starting edge node metrics retention cleanup",
		zap.Int("retention_days", retentionDays))

	count, err := h.metricsSvc.DeleteOldMetrics(ctx, retentionDays)
	if err != nil {
		zap.L().Error("metrics retention cleanup failed", zap.Error(err))
		return err
	}

	zap.L().Info("metrics retention cleanup completed",
		zap.Int64("deleted_count", count),
		zap.Int("retention_days", retentionDays))
	return nil
}
