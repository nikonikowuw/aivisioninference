package task

import (
	"context"
	"errors"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"
)

// handleAIVisionTaskPatrol 巡检所有 AI 视觉推理任务，并根据时间窗执行启停
func (h *Handler) handleAIVisionTaskPatrol(ctx context.Context, t *asynq.Task) error {
	zap.L().Info("starting aivision task patrol")
	if h.aiVisionTaskSvc == nil {
		zap.L().Warn("aiVisionTaskSvc is nil, skip patrol")
		return nil
	}

	err := h.aiVisionTaskSvc.PatrolTasks(ctx)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			zap.L().Info("aivision task patrol skipped because context is done", zap.Error(err))
			return nil
		}
		zap.L().Error("failed to patrol aivision tasks", zap.Error(err))
		return err
	}
	return nil
}
