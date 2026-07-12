package task

import (
	"context"
	"time"

	"github.com/hibiken/asynq"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"

	"github.com/niko-admin/niko-admin/internal/service"
)

// 计划任务相关 Asynq 任务类型
const (
	TypeEdgeScheduledTaskPatrol       = "edge_scheduled_task:patrol"
	TypeEdgeScheduledTaskCleanup      = "edge_scheduled_task:cleanup"
)

// EdgeScheduledTaskHandler Asynq 任务处理器，负责计划任务的定时巡检和记录清理
type EdgeScheduledTaskHandler struct {
	svc *service.EdgeScheduledTaskService
}

// NewEdgeScheduledTaskHandler 创建新的 EdgeScheduledTaskHandler
func NewEdgeScheduledTaskHandler(svc *service.EdgeScheduledTaskService) *EdgeScheduledTaskHandler {
	return &EdgeScheduledTaskHandler{svc: svc}
}

// RegisterHandlers 注册计划任务相关的 Asynq 处理器
func (h *EdgeScheduledTaskHandler) RegisterHandlers(mux *asynq.ServeMux) {
	mux.HandleFunc(TypeEdgeScheduledTaskPatrol, h.handlePatrol)
	mux.HandleFunc(TypeEdgeScheduledTaskCleanup, h.handleCleanup)
}

// RegisterPeriodic 注册计划任务的定时调度
func (h *EdgeScheduledTaskHandler) RegisterPeriodic(scheduler *asynq.Scheduler) {
	// 每分钟巡检一次，检查所有启用的计划任务是否匹配当前时间
	if _, err := scheduler.Register("@every 60s",
		asynq.NewTask(TypeEdgeScheduledTaskPatrol, nil)); err != nil {
		zap.L().Error("failed to register edge scheduled task patrol", zap.Error(err))
	}

	// 每天凌晨 3 点清理过期执行记录
	if _, err := scheduler.Register("0 3 * * *",
		asynq.NewTask(TypeEdgeScheduledTaskCleanup, nil)); err != nil {
		zap.L().Error("failed to register edge scheduled task cleanup", zap.Error(err))
	}
}

// handlePatrol 巡检所有启用的计划任务，对匹配当前时间的任务执行命令下发
func (h *EdgeScheduledTaskHandler) handlePatrol(ctx context.Context, t *asynq.Task) error {
	logger := zap.L().With(zap.String("task_type", TypeEdgeScheduledTaskPatrol))

	tasks, err := h.svc.ListEnabledTasks(ctx)
	if err != nil {
		logger.Error("failed to list enabled scheduled tasks", zap.Error(err))
		return err
	}

	now := time.Now()
	executedCount := 0

	for i := range tasks {
		task := &tasks[i]

		// 解析 cron 表达式，判断是否匹配当前时间
		schedule, err := cron.ParseStandard(task.CronExpr)
		if err != nil {
			logger.Warn("invalid cron expression",
				zap.String("task_id", task.ID),
				zap.String("task_name", task.Name),
				zap.String("cron_expr", task.CronExpr),
				zap.Error(err),
			)
			continue
		}

		// 检查当前时间是否匹配 cron：获取上次匹配后的下一触发时间，与当前时间比较
		nextTime := schedule.Next(now.Add(-2 * time.Minute))
		if nextTime.IsZero() || nextTime.After(now) {
			continue
		}

		// 执行任务（异步，不阻塞 patrol 循环）
		h.svc.ExecuteTask(context.Background(), task)
		executedCount++
	}

	if executedCount > 0 {
		logger.Info("edge scheduled task patrol completed",
			zap.Int("total_checked", len(tasks)),
			zap.Int("executed", executedCount),
		)
	}

	return nil
}

// handleCleanup 清理过期执行记录
func (h *EdgeScheduledTaskHandler) handleCleanup(ctx context.Context, t *asynq.Task) error {
	logger := zap.L().With(zap.String("task_type", TypeEdgeScheduledTaskCleanup))

	deleted, err := h.svc.CleanupRecords(ctx)
	if err != nil {
		logger.Error("failed to cleanup scheduled task records", zap.Error(err))
		return err
	}

	logger.Info("scheduled task records cleanup completed", zap.Int64("deleted", deleted))
	return nil
}
