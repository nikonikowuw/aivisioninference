package task

import (
	"context"
	"time"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"

	"github.com/niko-admin/niko-admin/internal/repository"
)

const (
	// TypeEdgeNodeMetricsCleanup 指标数据保留清理任务
	TypeEdgeNodeMetricsCleanup = "edge_node_metrics:cleanup"
)

// EdgeNodeMetricsHandler Asynq 处理器：指标数据保留清理
type EdgeNodeMetricsHandler struct {
	metricsRepo *repository.EdgeNodeMetricsRepository
}

// NewEdgeNodeMetricsHandler 创建 EdgeNodeMetricsHandler
func NewEdgeNodeMetricsHandler(metricsRepo *repository.EdgeNodeMetricsRepository) *EdgeNodeMetricsHandler {
	return &EdgeNodeMetricsHandler{metricsRepo: metricsRepo}
}

// RegisterHandlers 注册 Asynq 处理器
func (h *EdgeNodeMetricsHandler) RegisterHandlers(mux *asynq.ServeMux) {
	mux.HandleFunc(TypeEdgeNodeMetricsCleanup, h.handleCleanup)
}

// RegisterPeriodic 注册定时任务（每日凌晨3点清理）
func (h *EdgeNodeMetricsHandler) RegisterPeriodic(scheduler *asynq.Scheduler) {
	scheduler.Register("0 3 * * *", asynq.NewTask(TypeEdgeNodeMetricsCleanup, nil))
}

// handleCleanup 清理过期指标记录（默认保留7天）
func (h *EdgeNodeMetricsHandler) handleCleanup(ctx context.Context, t *asynq.Task) error {
	cutoff := time.Now().Add(-7 * 24 * time.Hour)
	count, err := h.metricsRepo.DeleteOlderThan(ctx, cutoff)
	if err != nil {
		zap.L().Error("metrics cleanup failed", zap.Error(err))
		return err
	}
	zap.L().Info("metrics cleanup completed",
		zap.Int64("deleted_count", count),
		zap.Time("cutoff", cutoff))
	return nil
}
