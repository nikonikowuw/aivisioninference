// Package task 提供基于 Asynq 的后台异步任务队列管理功能
package task

import (
	"time"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"

	"github.com/niko-admin/niko-admin/internal/service"
)

// NewServer 创建并配置一个新的 Asynq Server 用于消费和处理队列中的异步任务
func NewServer(rdb *redis.Client) *asynq.Server {
	opts := rdb.Options()
	// 使用 Redis 客户端的连接配置来初始化 Asynq Server
	return asynq.NewServer(asynq.RedisClientOpt{
		Addr:         opts.Addr,
		Password:     opts.Password,
		DB:           opts.DB,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	}, asynq.Config{
		Concurrency: 10, // 最大并行任务处理数
	})
}

// NewScheduler 创建并配置一个新的 Asynq Scheduler 用于定时任务调度
func NewScheduler(rdb *redis.Client) *asynq.Scheduler {
	opts := rdb.Options()
	return asynq.NewScheduler(asynq.RedisClientOpt{
		Addr:     opts.Addr,
		Password: opts.Password,
		DB:       opts.DB,
	}, &asynq.SchedulerOpts{
		Location: time.Local,
	})
}

// RegisterPeriodicTasks 注册所有定时任务到调度器
func RegisterPeriodicTasks(scheduler *asynq.Scheduler) {
	// 每 5 分钟检查一次设备状态
	scheduler.Register("*/5 * * * *", asynq.NewTask(TypeDeviceStatusCheck, nil))
}

// NewMux 创建并返回一个新的 Asynq ServeMux，并在此 Mux 上注册所有任务处理 Handler
func NewMux(mailSvc *service.MailService, deviceStatusHandler *DeviceStatusHandler) *asynq.ServeMux {
	mux := asynq.NewServeMux()
	// 初始化 Handler 并注册其路由
	NewHandler(mailSvc, deviceStatusHandler).RegisterHandlers(mux)
	return mux
}
