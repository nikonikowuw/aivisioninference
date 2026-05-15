// Package main is the entry point for the niko-admin server.
//
//	@title           Niko Admin API
//	@version         1.0
//	@description     基于 Gin 的后台管理系统 API
//	@host            localhost:8080
//	@BasePath        /api/v1
//	@securityDefinitions.apikey BearerAuth
//	@in              header
//	@name            Authorization
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"github.com/niko-admin/niko-admin/internal/config"
	"github.com/niko-admin/niko-admin/internal/pkg/cache"
	"github.com/niko-admin/niko-admin/internal/pkg/database"
	"github.com/niko-admin/niko-admin/internal/pkg/jwt"
	"github.com/niko-admin/niko-admin/internal/pkg/ws"
	"github.com/niko-admin/niko-admin/internal/router"
)

var (
	Version   = "dev"
	BuildTime = "unknown"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Initialize logger
	initLogger(cfg.Log)

	zap.L().Info("starting niko-admin",
		zap.String("version", Version),
		zap.String("build_time", BuildTime),
		zap.String("env", cfg.App.Env),
	)

	// Connect to PostgreSQL
	db, err := database.New(
		fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
			cfg.DB.Host, cfg.DB.Port, cfg.DB.User, cfg.DB.Password, cfg.DB.Name),
		cfg.DB.MaxOpenConns,
		cfg.DB.MaxIdleConns,
	)
	if err != nil {
		zap.L().Fatal("failed to connect to database", zap.Error(err))
	}
	zap.L().Info("database connected")

	// Connect to Redis
	rdb, err := cache.New(cfg.Redis.Host, cfg.Redis.Port, cfg.Redis.Password, cfg.Redis.DB)
	if err != nil {
		zap.L().Fatal("failed to connect to redis", zap.Error(err))
	}
	zap.L().Info("redis connected")

	// Initialize JWT manager
	jwtManager := jwt.NewManager(
		cfg.JWT.Secret,
		cfg.JWT.Issuer,
		cfg.JWT.Audience,
		cfg.JWT.AccessExpireSec,
		cfg.JWT.RefreshExpireSec,
		rdb,
	)

	// Initialize WebSocket hub
	hub := ws.NewHub()
	go hub.Run()

	// Create router
	routerCfg := &router.Config{
		AppEnv:            cfg.App.Env,
		AllowOrigins:      cfg.CORS.AllowOrigins,
		RequestsPerMinute: cfg.RateLimit.RequestsPerMinute,
	}
	r := router.New(db, rdb, jwtManager, hub, routerCfg)

	// Start HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.App.Port),
		Handler:      r.Engine(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start Asynq server in background
	asynqServer := router.NewAsynqServer(rdb)
	asynqMux := router.NewAsynqMux()
	go func() {
		zap.L().Info("starting asynq server")
		if err := asynqServer.Run(asynqMux); err != nil {
			zap.L().Error("asynq server error", zap.Error(err))
		}
	}()

	// Graceful shutdown
	go func() {
		zap.L().Info("server starting", zap.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			zap.L().Fatal("server listen error", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	zap.L().Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		zap.L().Error("server forced shutdown", zap.Error(err))
	}

	asynqServer.Shutdown()

	zap.L().Info("server exited")
}

func initLogger(cfg config.LogConfig) {
	level := zap.InfoLevel
	if cfg.Level == "debug" {
		level = zap.DebugLevel
	}

	var zapCfg zap.Config
	if cfg.Format == "json" {
		zapCfg = zap.NewProductionConfig()
	} else {
		zapCfg = zap.NewDevelopmentConfig()
	}
	zapCfg.Level = zap.NewAtomicLevelAt(level)

	logger, err := zapCfg.Build()
	if err != nil {
		log.Fatalf("failed to init logger: %v", err)
	}
	zap.ReplaceGlobals(logger)
}
