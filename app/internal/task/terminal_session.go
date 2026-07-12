// Package task 提供基于 Asynq 的后台异步任务队列管理功能。
package task

import (
	"context"
	"time"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"

	"github.com/niko-admin/niko-admin/internal/repository"
	"github.com/niko-admin/niko-admin/internal/service"
)

// 终端会话相关 Asynq 任务类型
const (
	TypeTerminalSessionCleanup = "terminal_session:cleanup"
)

// TerminalSessionHandler 终端会话 Asynq 任务处理器
type TerminalSessionHandler struct {
	sessionRepo *repository.TerminalSessionRepository
	svc         *service.TerminalSessionService
}

// NewTerminalSessionHandler 创建 TerminalSessionHandler
func NewTerminalSessionHandler(
	sessionRepo *repository.TerminalSessionRepository,
	svc *service.TerminalSessionService,
) *TerminalSessionHandler {
	return &TerminalSessionHandler{
		sessionRepo: sessionRepo,
		svc:         svc,
	}
}

// RegisterHandlers 注册终端会话相关的 Asynq 处理器
func (h *TerminalSessionHandler) RegisterHandlers(mux *asynq.ServeMux) {
	mux.HandleFunc(TypeTerminalSessionCleanup, h.handleCleanup)
}

// RegisterPeriodic 注册终端会话的定时任务
func (h *TerminalSessionHandler) RegisterPeriodic(scheduler *asynq.Scheduler) {
	// 每 5 分钟清理超时的暂停会话
	if _, err := scheduler.Register("@every 5m",
		asynq.NewTask(TypeTerminalSessionCleanup, nil)); err != nil {
		zap.L().Error("failed to register terminal session cleanup", zap.Error(err))
	}
}

// handleCleanup 清理暂停超过 5 分钟的终端会话
func (h *TerminalSessionHandler) handleCleanup(ctx context.Context, t *asynq.Task) error {
	timeout := 5 * time.Minute
	sessions, err := h.sessionRepo.ListPausedBefore(ctx, timeout)
	if err != nil {
		return err
	}

	for _, session := range sessions {
		if err := h.svc.CloseSession(ctx, session.ID, "timeout"); err != nil {
			zap.L().Warn("failed to close timed out session",
				zap.String("session_id", session.ID),
				zap.Error(err),
			)
		}
	}

	zap.L().Info("terminal session cleanup completed",
		zap.Int("closed_count", len(sessions)),
	)
	return nil
}
