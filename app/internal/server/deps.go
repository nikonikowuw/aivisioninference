// Package server 提供应用级依赖组装和服务生命周期管理。
package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/niko-admin/niko-admin/internal/config"
	"github.com/niko-admin/niko-admin/internal/pkg/cache"
	"github.com/niko-admin/niko-admin/internal/pkg/database"
	"github.com/niko-admin/niko-admin/internal/pkg/jwt"
	applog "github.com/niko-admin/niko-admin/internal/pkg/log"
	"github.com/niko-admin/niko-admin/internal/router"
)

func provideLoggers(cfg *config.Config) *applog.Logger {
	return applog.Init(cfg.Log)
}

func provideAccessLogger(l *applog.Logger) *zap.Logger {
	return l.Access
}

func provideDB(cfg *config.Config) (*gorm.DB, error) {
	return database.New(
		fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			cfg.DB.Host, cfg.DB.Port, cfg.DB.User, cfg.DB.Password, cfg.DB.Name, cfg.DB.SSLMode),
		cfg.DB.MaxOpenConns,
		cfg.DB.MaxIdleConns,
	)
}

func provideRedis(cfg *config.Config) (*redis.Client, error) {
	return cache.New(cfg.Redis.Host, cfg.Redis.Port, cfg.Redis.Password, cfg.Redis.DB)
}

func provideJWTManager(cfg *config.Config, rdb *redis.Client) *jwt.Manager {
	return jwt.NewManager(
		cfg.JWT.Secret, cfg.JWT.Issuer, cfg.JWT.Audience,
		cfg.JWT.AccessExpireSec, cfg.JWT.RefreshExpireSec, rdb,
	)
}

func provideRouterConfig(cfg *config.Config) *router.Config {
	return &router.Config{
		AppEnv:                    cfg.App.Env,
		AllowOrigins:              cfg.CORS.AllowOrigins,
		RequestsPerMinute:         cfg.RateLimit.RequestsPerMinute,
		TrustedProxies:            cfg.Proxy.TrustedProxies,
		PermissionTreeRedisEnable: true,
		ChunkSizeMB:               cfg.Storage.ChunkSizeMB,
		MaxFileSizeMB:             cfg.Storage.MaxFileSizeMB,
		LocalUploadDir:            cfg.Storage.Local.UploadDir,
		LocalPublicURL:            cfg.Storage.Local.PublicURL,
		ZLMAPIURL:                 cfg.ZLM.APIURL,
		ZLMSecret:                 cfg.ZLM.Secret,
	}
}

func provideHTTPServer(r *router.Router, cfg *config.Config) *http.Server {
	return &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.App.Port),
		Handler:      r.Engine(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
}

func provideAsynqMux(db *gorm.DB, rdb *redis.Client, cfg *router.Config) *asynq.ServeMux {
	return router.NewAsynqMux(db, rdb, cfg)
}

func provideAsynqScheduler(rdb *redis.Client) *asynq.Scheduler {
	return router.NewAsynqScheduler(rdb)
}
