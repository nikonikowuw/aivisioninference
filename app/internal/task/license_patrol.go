// Package task 提供异步后台任务处理，包含授权过期巡检等定时任务。
package task

import (
	"context"
	"time"

	"go.uber.org/zap"
)

// LicenseExpirer 授权过期处理接口
type LicenseExpirer interface {
	// RefreshExpiredStatus 巡检并刷新过期授权状态，返回过期的授权 ID 列表
	RefreshExpiredStatus(ctx context.Context) []string
}

// TaskStopper 推理任务停止器接口（通过 IPC 向 C++ 引擎下发停止指令）
type TaskStopper interface {
	// StopRunningByExpiredLicense 停止因授权过期而需要终止的运行中任务
	StopRunningByExpiredLicense(ctx context.Context, expiredLicenseIDs []string) error
}

// LicensePatrolWorker 授权过期巡检 Worker，定期检查授权有效期并强制终止过期任务。
type LicensePatrolWorker struct {
	licenseExpirer LicenseExpirer
	taskStopper    TaskStopper
	interval       time.Duration
	cancel         context.CancelFunc
}

// NewLicensePatrolWorker 创建授权巡检 Worker 实例
func NewLicensePatrolWorker(licenseExpirer LicenseExpirer, taskStopper TaskStopper, interval time.Duration) *LicensePatrolWorker {
	if interval <= 0 {
		interval = 5 * time.Minute // 默认每 5 分钟巡检一次
	}
	return &LicensePatrolWorker{
		licenseExpirer: licenseExpirer,
		taskStopper:    taskStopper,
		interval:       interval,
	}
}

// Start 启动后台巡检 goroutine
func (w *LicensePatrolWorker) Start(ctx context.Context) {
	ctx, w.cancel = context.WithCancel(ctx)
	go w.run(ctx)
	zap.L().Info("license patrol worker started", zap.Duration("interval", w.interval))
}

// Stop 停止巡检
func (w *LicensePatrolWorker) Stop() {
	if w.cancel != nil {
		w.cancel()
		zap.L().Info("license patrol worker stopped")
	}
}

// run 巡检主循环
func (w *LicensePatrolWorker) run(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.patrol(ctx)
		}
	}
}

// patrol 执行一次巡检：检查过期授权并终止相关运行中任务
func (w *LicensePatrolWorker) patrol(ctx context.Context) {
	// 1. 刷新过期授权状态
	expiredIDs := w.licenseExpirer.RefreshExpiredStatus(ctx)
	if len(expiredIDs) == 0 {
		return
	}

	zap.L().Info("license patrol detected expired licenses",
		zap.Int("count", len(expiredIDs)),
		zap.Strings("license_ids", expiredIDs))

	// 2. 停止相关运行中任务
	if w.taskStopper != nil {
		if err := w.taskStopper.StopRunningByExpiredLicense(ctx, expiredIDs); err != nil {
			zap.L().Error("failed to stop tasks for expired licenses", zap.Error(err))
		}
	}
}
