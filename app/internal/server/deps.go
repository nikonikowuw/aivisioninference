// Package server 提供应用级依赖组装和服务生命周期管理。
package server

import (
	"fmt"
	"net/http"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/niko-admin/niko-admin/internal/config"
	"github.com/niko-admin/niko-admin/internal/pkg/cache"
	"github.com/niko-admin/niko-admin/internal/pkg/database"
	"github.com/niko-admin/niko-admin/internal/pkg/jwt"
	applog "github.com/niko-admin/niko-admin/internal/pkg/log"
	"github.com/niko-admin/niko-admin/internal/pkg/mqttsync"
	"github.com/niko-admin/niko-admin/internal/pkg/ws"
	"github.com/niko-admin/niko-admin/internal/router"
	"github.com/niko-admin/niko-admin/internal/service"
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
	e := cfg.Engine
	return &router.Config{
		AppEnv:                    cfg.App.Env,
		AllowOrigins:              cfg.CORS.AllowOrigins,
		RequestsPerMinute:         cfg.RateLimit.RequestsPerMinute,
		TrustedProxies:            cfg.Proxy.TrustedProxies,
		PermissionTreeRedisEnable: true,
		ChunkSizeMB:               cfg.Storage.ChunkSizeMB,
		MaxFileSizeMB:             cfg.Storage.MaxFileSizeMB,
		MaxAlgoFileSizeMB:         cfg.Storage.MaxAlgoFileSizeMB,
		MaxUploadConcurrency:      cfg.Storage.MaxUploadConcurrency,
		LocalUploadDir:            cfg.Storage.Local.UploadDir,
		LocalPublicURL:            cfg.Storage.Local.PublicURL,
		ZLMAPIURL:                 cfg.ZLM.APIURL,
		ZLMSecret:                 cfg.ZLM.Secret,
		Engine: router.RouterEngineConfig{
			MinCompatibleVersion:      e.MinCompatibleVersion,
			VersionCheckEnabled:       e.VersionCheckEnabled,
			HeartbeatTimeoutSec:       e.HeartbeatTimeoutSec,
			HeartbeatCheckIntervalSec: e.HeartbeatCheckInterval,
		},
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

func provideAsynqMux(db *gorm.DB, rdb *redis.Client, cfg *router.Config, client mqtt.Client, scheduler *asynq.Scheduler, hub *ws.Hub, runtimeStateStore *service.EdgeNodeRuntimeStateStore) *asynq.ServeMux {
	syncManager := mqttsync.NewMqttSyncManager(rdb)
	return router.NewAsynqMux(db, rdb, cfg, client, syncManager, scheduler, hub, runtimeStateStore)
}

func provideRuntimeStateStore(r *router.Router) *service.EdgeNodeRuntimeStateStore {
	return r.EdgeNodeSvc.RuntimeStateStore()
}

func provideAsynqScheduler(rdb *redis.Client) *asynq.Scheduler {
	return router.NewAsynqScheduler(rdb)
}

func provideMqttClient(cfg *config.Config) (mqtt.Client, error) {
	opts := mqtt.NewClientOptions()
	scheme := "tcp"
	if cfg.MQTT.UseTLS {
		scheme = "ssl"
	}
	broker := fmt.Sprintf("%s://%s:%d", scheme, cfg.MQTT.Host, cfg.MQTT.Port)
	opts.AddBroker(broker)
	opts.SetClientID(cfg.MQTT.ClientID)
	if cfg.MQTT.Username != "" {
		opts.SetUsername(cfg.MQTT.Username)
	}
	if cfg.MQTT.Password != "" {
		opts.SetPassword(cfg.MQTT.Password)
	}
	opts.SetAutoReconnect(true)
	opts.SetCleanSession(false)

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return nil, fmt.Errorf("MQTT connection failed: %w", token.Error())
	}

	zap.L().Info("MQTT: connected to broker", zap.String("broker", broker))
	return client, nil
}

func provideMqttServer(client mqtt.Client, r *router.Router) *MqttServer {
	return NewMqttServer(client, r.MqttMux, r.EdgeMqttHandler)
}

func provideEdgeNodeService(r *router.Router) *service.EdgeNodeService {
	return r.EdgeNodeSvc
}
