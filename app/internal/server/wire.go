//go:build wireinject
// +build wireinject

// Package server 提供应用级依赖组装和服务生命周期管理。
package server

import (
	"github.com/google/wire"

	"github.com/niko-admin/niko-admin/internal/config"
	"github.com/niko-admin/niko-admin/internal/pkg/ws"
	"github.com/niko-admin/niko-admin/internal/router"
)

// InitializeApp 使用 Wire 构建完整的应用依赖图。
func InitializeApp() (*App, error) {
	wire.Build(
		config.Load,
		provideLoggers,
		provideAccessLogger,
		provideDB,
		provideRedis,
		provideJWTManager,
		ws.NewHub,
		provideRouterConfig,
		router.New,
		provideHTTPServer,
		router.NewAsynqServer,
		router.NewAsynqScheduler,
		provideAsynqMux,
		wire.Struct(new(App), "*"),
	)
	return nil, nil
}
