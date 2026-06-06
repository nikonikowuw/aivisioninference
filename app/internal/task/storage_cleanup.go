// Package task 提供基于 Asynq 的后台异步任务队列管理功能
package task

import (
	"context"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"

	"github.com/niko-admin/niko-admin/internal/service"
)

// 任务类型常量
const (
	TypeStorageCleanupCron      = "storage:cleanup:cron"
	TypeStorageCleanupThreshold = "storage:cleanup:threshold"
)

// CronCleanupHandler 处理 CRON 定时清理任务
type CronCleanupHandler struct {
	storageSvc *service.StorageService
}

// ThresholdCleanupHandler 处理阈值检查+清理任务
type ThresholdCleanupHandler struct {
	storageSvc *service.StorageService
}

func NewCronCleanupHandler(storageSvc *service.StorageService) *CronCleanupHandler {
	return &CronCleanupHandler{storageSvc: storageSvc}
}

func NewThresholdCleanupHandler(storageSvc *service.StorageService) *ThresholdCleanupHandler {
	return &ThresholdCleanupHandler{storageSvc: storageSvc}
}

// RegisterHandlers 注册清理任务 handler 到 mux
func (h *CronCleanupHandler) RegisterHandlers(mux *asynq.ServeMux) {
	mux.HandleFunc(TypeStorageCleanupCron, h.handle)
}

func (h *ThresholdCleanupHandler) RegisterHandlers(mux *asynq.ServeMux) {
	mux.HandleFunc(TypeStorageCleanupThreshold, h.handle)
}

// handle 处理 CRON 定时清理：直接执行清理（Phase 1 过期清理）
func (h *CronCleanupHandler) handle(ctx context.Context, _ *asynq.Task) error {
	cfg, err := h.storageSvc.GetConfig()
	if err != nil {
		return err
	}
	if !cfg.Enabled || cfg.CleanupMode != service.CleanupModeCron {
		return nil
	}

	zap.L().Info("cron cleanup triggered", zap.String("cron", cfg.CronExpression))
	start := time.Now()
	if err := h.storageSvc.RunCleanup(); err != nil {
		zap.L().Error("cron cleanup failed", zap.Error(err), zap.Duration("duration", time.Since(start)))
		return err
	}
	zap.L().Info("cron cleanup completed", zap.Duration("duration", time.Since(start)))
	return nil
}

// handle 处理阈值检查：检查磁盘使用率，超阈值则执行清理（Phase 1 + Phase 2）
func (h *ThresholdCleanupHandler) handle(ctx context.Context, _ *asynq.Task) error {
	cfg, err := h.storageSvc.GetConfig()
	if err != nil {
		return err
	}
	if !cfg.Enabled || cfg.CleanupMode != service.CleanupModeThreshold {
		return nil
	}

	overThreshold, err := h.storageSvc.IsOverThreshold()
	if err != nil {
		zap.L().Warn("failed to check disk threshold", zap.Error(err))
		return err
	}
	if !overThreshold {
		return nil
	}

	zap.L().Info("disk usage over threshold, triggering cleanup",
		zap.Float64("threshold", cfg.ThresholdValue),
		zap.Float64("target", cfg.TargetPercentage))
	start := time.Now()
	if err := h.storageSvc.RunCleanup(); err != nil {
		zap.L().Error("threshold cleanup failed", zap.Error(err), zap.Duration("duration", time.Since(start)))
		return err
	}
	zap.L().Info("threshold cleanup completed", zap.Duration("duration", time.Since(start)))
	return nil
}

// StorageScheduler 管理存储清理的 CRON 调度
// 根据 CleanupMode 动态注册/删除 Asynq CRON 任务
type StorageScheduler struct {
	scheduler        *asynq.Scheduler
	storageSvc       *service.StorageService
	cronEntryID      string // cron 模式的调度 entry
	thresholdEntryID string // threshold 模式的调度 entry
}

func NewStorageScheduler(scheduler *asynq.Scheduler, storageSvc *service.StorageService) *StorageScheduler {
	return &StorageScheduler{
		scheduler:  scheduler,
		storageSvc: storageSvc,
	}
}

// Init 初始化调度：根据当前配置注册对应的 CRON 任务
// 先清除已有的调度条目，防止进程重启后重复注册
func (ss *StorageScheduler) Init() error {
	ss.clearSchedule()
	cfg, err := ss.storageSvc.GetConfig()
	if err != nil {
		return fmt.Errorf("failed to get storage config for scheduler init: %w", err)
	}
	if !cfg.Enabled {
		return nil
	}
	return ss.syncSchedule(cfg)
}

// UpdateConfig 更新调度：删除旧任务，注册新任务
func (ss *StorageScheduler) UpdateConfig(cfg *service.StorageConfig) error {
	// 先清除所有已注册的调度
	ss.clearSchedule()
	if !cfg.Enabled {
		return nil
	}
	return ss.syncSchedule(cfg)
}

// syncSchedule 根据配置注册对应的 CRON 任务
func (ss *StorageScheduler) syncSchedule(cfg *service.StorageConfig) error {
	switch cfg.CleanupMode {
	case service.CleanupModeCron:
		entryID, err := ss.scheduler.Register(cfg.CronExpression, asynq.NewTask(TypeStorageCleanupCron, nil))
		if err != nil {
			return fmt.Errorf("failed to register cron cleanup task: %w", err)
		}
		ss.cronEntryID = entryID
		zap.L().Info("registered cron cleanup schedule", zap.String("cron", cfg.CronExpression))

	case service.CleanupModeThreshold:
		cronExpr := cfg.ThresholdCheckCron
		if cronExpr == "" {
			cronExpr = "*/5 * * * *"
		}
		entryID, err := ss.scheduler.Register(cronExpr, asynq.NewTask(TypeStorageCleanupThreshold, nil))
		if err != nil {
			return fmt.Errorf("failed to register threshold cleanup task: %w", err)
		}
		ss.thresholdEntryID = entryID
		zap.L().Info("registered threshold cleanup schedule", zap.String("cron", cronExpr))
	}
	return nil
}

// clearSchedule 清除所有已注册的调度
func (ss *StorageScheduler) clearSchedule() {
	if ss.cronEntryID != "" {
		ss.scheduler.Unregister(ss.cronEntryID)
		ss.cronEntryID = ""
	}
	if ss.thresholdEntryID != "" {
		ss.scheduler.Unregister(ss.thresholdEntryID)
		ss.thresholdEntryID = ""
	}
}
