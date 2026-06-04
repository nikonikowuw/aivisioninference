// Package server 提供应用级依赖组装和服务生命周期管理。
package server

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"

	applog "github.com/niko-admin/niko-admin/internal/pkg/log"
	validatorx "github.com/niko-admin/niko-admin/internal/pkg/validator"
	"github.com/niko-admin/niko-admin/internal/pkg/ws"
)

// App 聚合应用启动所需的所有依赖。
type App struct {
	Loggers        *applog.Logger
	HTTPServer     *http.Server
	AsynqServer    *asynq.Server
	AsynqMux       *asynq.ServeMux
	AsynqScheduler *asynq.Scheduler
	Hub            *ws.Hub
}

// Run 启动各个服务器组件并等待退出信号。
func (a *App) Run() {
	defer a.Loggers.Sync()

	// 必须在 zap.L() 就绪后初始化
	if err := validatorx.InitGinBindingValidator(); err != nil {
		zap.L().Fatal("failed to initialize gin binding validator", zap.Error(err))
	}

	// 启动 WebSocket Hub
	go a.Hub.Run()

	// 启动 Asynq 后台 worker
	go func() {
		zap.L().Info("starting asynq server")
		if err := a.AsynqServer.Run(a.AsynqMux); err != nil {
			zap.L().Error("asynq server error", zap.Error(err))
		}
	}()

	// 启动 Asynq 定时任务调度器
	go func() {
		zap.L().Info("starting asynq scheduler")
		if err := a.AsynqScheduler.Run(); err != nil {
			zap.L().Error("asynq scheduler error", zap.Error(err))
		}
	}()

	// 启动 HTTP 服务
	go func() {
		zap.L().Info("server starting", zap.String("addr", a.HTTPServer.Addr))
		if err := a.HTTPServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			zap.L().Fatal("server listen error", zap.Error(err))
		}
	}()

	// 等待退出信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	zap.L().Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := a.HTTPServer.Shutdown(ctx); err != nil {
		zap.L().Error("server forced shutdown", zap.Error(err))
	}
	// 停止定时任务调度器
	if a.AsynqScheduler != nil {
		a.AsynqScheduler.Shutdown()
	}
	a.AsynqServer.Shutdown()
	zap.L().Info("server exited")
}
