// Package router 提供 HTTP 路由注册和依赖注入编排，串联 Handler、Service、Repository 各层。
package router

import (
	"net/http"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/niko-admin/niko-admin/internal/handler"
	"github.com/niko-admin/niko-admin/internal/middleware"
	"github.com/niko-admin/niko-admin/internal/pkg/cache"
	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
	"github.com/niko-admin/niko-admin/internal/pkg/httpx"
	"github.com/niko-admin/niko-admin/internal/pkg/jwt"
	"github.com/niko-admin/niko-admin/internal/pkg/mqttmux"
	"github.com/niko-admin/niko-admin/internal/pkg/mqttsync"
	"github.com/niko-admin/niko-admin/internal/pkg/response"
	"github.com/niko-admin/niko-admin/internal/pkg/ws"
	"github.com/niko-admin/niko-admin/internal/repository"
	"github.com/niko-admin/niko-admin/internal/service"
	"github.com/niko-admin/niko-admin/internal/task"
)

// Router holds all dependencies for route registration.
type Router struct {
	engine             *gin.Engine
	db                 *gorm.DB
	rdb                *redis.Client
	jwtManager         *jwt.Manager
	hub                *ws.Hub
	config             *Config
	accessLogger       *zap.Logger
	scheduler          *asynq.Scheduler
	rbacCache          cache.Cache
	SIPRuntimeSvc      *service.SIPRuntimeService
	mqttClient         mqtt.Client
	MqttMux            *mqttmux.Mux
	EdgeMqttHandler    *handler.EdgeMqttHandler
	EdgeNodeSvc        *service.EdgeNodeService
	EngineMetricsStore *service.EngineMetricsStore
}

// RBAC returns the RBAC middleware, bound to the Router's cached dependencies.
func (r *Router) RBAC() gin.HandlerFunc {
	return middleware.RBAC(r.rbacCache, r.db)
}

// Config holds router-level configuration.
type Config struct {
	AppEnv                    string
	AllowOrigins              []string
	RequestsPerMinute         int
	TrustedProxies            []string
	PermissionTreeRedisEnable bool
	ChunkSizeMB               int
	MaxFileSizeMB             int64
	MaxAlgoFileSizeMB         int64
	MaxUploadConcurrency      int
	LocalUploadDir            string
	LocalPublicURL            string
	// Storage 存储配置
	StorageType  string `yaml:"storage_type" mapstructure:"storage_type"` // "local" 或 "oss"
	OSSEndpoint  string `yaml:"oss_endpoint" mapstructure:"oss_endpoint"`
	OSSAccessKey string `yaml:"oss_access_key" mapstructure:"oss_access_key"`
	OSSSecretKey string `yaml:"oss_secret_key" mapstructure:"oss_secret_key"`
	OSSBucket    string `yaml:"oss_bucket" mapstructure:"oss_bucket"`
	OSSUseSSL    bool   `yaml:"oss_use_ssl" mapstructure:"oss_use_ssl"`
	OSSDomain    string `yaml:"oss_domain" mapstructure:"oss_domain"`
	ZLMAPIURL    string
	ZLMSecret    string
	// GB28181 GB/T 28181 配置
	GB28181Enabled        bool          `yaml:"gb28181_enabled" mapstructure:"gb28181_enabled"`
	GB28181Domain         string        `yaml:"gb28181_domain" mapstructure:"gb28181_domain"`
	GB28181Password       string        `yaml:"gb28181_password" mapstructure:"gb28181_password"`
	GB28181HeartbeatSec   int           `yaml:"gb28181_heartbeat_sec" mapstructure:"gb28181_heartbeat_sec"`
	GB28181CatalogSyncSec int           `yaml:"gb28181_catalog_sync_sec" mapstructure:"gb28181_catalog_sync_sec"`
	GB28181PlaybackMaxSec int           `yaml:"gb28181_playback_max_sec" mapstructure:"gb28181_playback_max_sec"`
	GB28181StreamTimeout  time.Duration `yaml:"gb28181_stream_timeout" mapstructure:"gb28181_stream_timeout"`
	// ZLM 端口配置（可被 docker 端口映射覆盖）
	ZLMRTMPPort int `yaml:"zlm_rtmp_port" mapstructure:"zlm_rtmp_port"`
	ZLMRTSPPort int `yaml:"zlm_rtsp_port" mapstructure:"zlm_rtsp_port"`
	ZLMHTTPPort int `yaml:"zlm_http_port" mapstructure:"zlm_http_port"`
	// ZLM 实际对外 IP（设备推流目标地址，默认从 ZLMAPIURL 解析）
	ZLMExternalIP string `yaml:"zlm_external_ip" mapstructure:"zlm_external_ip"`
	Engine        RouterEngineConfig
}

type RouterEngineConfig struct {
	MinCompatibleVersion      string
	VersionCheckEnabled       bool
	HeartbeatTimeoutSec       int
	HeartbeatCheckIntervalSec int
}

// New creates a new Router with all dependencies wired.
func New(db *gorm.DB, rdb *redis.Client, jwtManager *jwt.Manager, hub *ws.Hub, cfg *Config, accessLogger *zap.Logger, scheduler *asynq.Scheduler, mqttClient mqtt.Client) *Router {
	engine := gin.New()

	httpx.TrustedProxies = cfg.TrustedProxies

	r := &Router{
		engine:       engine,
		db:           db,
		rdb:          rdb,
		jwtManager:   jwtManager,
		hub:          hub,
		config:       cfg,
		accessLogger: accessLogger,
		scheduler:    scheduler,
		mqttClient:   mqttClient,
	}

	r.setupMiddleware()
	r.setupRoutes()

	return r
}

// Engine returns the underlying gin.Engine.
func (r *Router) Engine() *gin.Engine {
	return r.engine
}

func (r *Router) setupMiddleware() {
	// Global middleware
	r.engine.Use(middleware.Recovery())
	r.engine.Use(middleware.Logger(r.accessLogger))
	r.engine.Use(middleware.I18n())
	r.engine.Use(middleware.CORS(r.config.AllowOrigins))
	if r.config.RequestsPerMinute > 0 {
		r.engine.Use(middleware.RateLimit(r.rdb, r.config.RequestsPerMinute))
	}

	r.engine.Use(middleware.ErrorHandler())
}

func (r *Router) setupRoutes() {
	v1 := r.engine.Group("/api/v1")

	deps, err := InitializeRouteDeps(r.db, r.rdb, r.jwtManager, r.hub, r.config, r.scheduler, r.mqttClient)
	if err != nil {
		zap.L().Fatal("initialize route dependencies failed", zap.Error(err))
	}
	r.SIPRuntimeSvc = deps.SIPRuntimeSvc
	r.rbacCache = deps.RBACCache
	r.EdgeNodeSvc = deps.EdgeNodeSvc
	r.EngineMetricsStore = deps.EngineMetricsStore
	r.EdgeMqttHandler = deps.EdgeMqttHandler
	r.MqttMux = deps.MqttMux

	// Auth (no auth required)
	r.registerAuthRoutes(v1, deps)

	// WebSocket
	v1.GET("/ws", deps.WSHandler.HandleWebSocket)

	// Algorithm Package Download (no auth required, verified by short-lived token)
	v1.GET("/internal/algo/download", deps.AlgorithmPackageHandler.DownloadAlgorithmPackage)

	// Protected routes
	authorized := v1.Group("")
	authorized.Use(middleware.Auth(r.jwtManager))
	authorized.Use(middleware.Audit(deps.AuditService))

	r.registerUserRoutes(authorized, deps)
	r.registerRoleRoutes(authorized, deps)
	r.registerPermissionRoutes(authorized, deps)
	r.registerFileRoutes(authorized, deps)
	r.registerAlgoPackageRoutes(authorized, deps)
	r.registerAuditRoutes(authorized, deps)
	r.registerGeneralTaskRoutes(authorized, deps)

	// AIVisionTasks
	RegisterAIVisionTaskRoutes(authorized, deps.AIVisionTaskHandler, r.RBAC())

	// AI Time Schedules (reusable time configurations for AI tasks)
	RegisterAITimeScheduleRoutes(authorized, deps.AITimeScheduleHandler, r.RBAC())

	r.registerSystemRoutes(authorized, v1, deps)
	r.registerFeedbackRoutes(authorized, deps)
	r.registerDeviceRoutes(authorized, deps)
	r.registerMediaRoutes(authorized, v1, deps)
	r.registerPersonRoutes(authorized, deps)
	r.registerSmartRecordRoutes(authorized, deps)
	r.registerEdgeNodeRoutes(authorized, v1, deps)

	r.engine.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api") {
			response.Err(c, apperrors.New(apperrors.ErrNotFound, "资源不存在"))
			return
		}
		c.File("./web/dist/index.html")
	})

	// Swagger UI (non-production only)
	if r.config.AppEnv != "prod" {
		r.engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	// Health check
	r.engine.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Static file serving for uploaded files
	uploads := r.engine.Group("/uploads")
	uploads.Use(middleware.CORS(r.config.AllowOrigins))
	uploads.Use(func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("Cache-Control", "public, max-age=31536000")
		c.Next()
	})
	uploads.Static("", "uploads")

	// Frontend static files (SPA)
	r.engine.Static("/assets", "./web/dist/assets")
	r.engine.StaticFile("/favicon.ico", "./web/dist/favicon.ico")
}

func (r *Router) registerAuthRoutes(v1 *gin.RouterGroup, deps *RouteDeps) {
	authHandler := deps.AuthHandler
	v1.POST("/auth/login", authHandler.Login)
	v1.POST("/auth/password-reset/request", authHandler.RequestPasswordReset)
	v1.POST("/auth/password-reset/confirm", authHandler.ResetPassword)
	v1.POST("/auth/refresh", authHandler.Refresh)
	v1.POST("/auth/logout", middleware.Auth(r.jwtManager), middleware.Audit(deps.AuditService), authHandler.Logout)
	v1.GET("/auth/me", middleware.Auth(r.jwtManager), authHandler.Me)
	v1.PUT("/auth/password", middleware.Auth(r.jwtManager), authHandler.ChangePassword)
	v1.PUT("/auth/profile", middleware.Auth(r.jwtManager), authHandler.UpdateProfile)
	v1.POST("/auth/avatar", middleware.Auth(r.jwtManager), authHandler.UploadAvatar)
}

func (r *Router) registerUserRoutes(authorized *gin.RouterGroup, deps *RouteDeps) {
	userHandler := deps.UserHandler
	users := authorized.Group("/users")
	users.Use(r.RBAC())
	{
		users.GET("", userHandler.List)
		users.POST("", userHandler.Create)
		users.GET("/export", userHandler.ExportCSV)
		users.POST("/import", userHandler.ImportCSV)
		users.POST("/batch-delete", userHandler.BatchDelete)
		users.PUT("/batch-status", userHandler.BatchUpdateStatus)
		users.GET("/:id", userHandler.GetByID)
		users.PUT("/:id", userHandler.Update)
		users.DELETE("/:id", userHandler.Delete)
		users.PUT("/:id/password", userHandler.ResetPassword)
		users.POST("/:id/avatar", userHandler.UploadAvatar)
	}
}

func (r *Router) registerRoleRoutes(authorized *gin.RouterGroup, deps *RouteDeps) {
	roleHandler := deps.RoleHandler

	// 例外：无需 RBAC 的公开路由（所有认证用户可查看角色列表）
	authorized.GET("/roles", roleHandler.List)

	// 需要 RBAC 的管理路由
	roleMgmt := authorized.Group("/roles")
	roleMgmt.Use(r.RBAC())
	{
		roleMgmt.POST("", roleHandler.Create)
		roleMgmt.GET("/export", roleHandler.ExportCSV)
		roleMgmt.POST("/batch-delete", roleHandler.BatchDelete)
		roleMgmt.GET("/:id", roleHandler.GetByID)
		roleMgmt.PUT("/:id", roleHandler.Update)
		roleMgmt.DELETE("/:id", roleHandler.Delete)
		roleMgmt.GET("/:id/permissions", roleHandler.GetPermissions)
		roleMgmt.PUT("/:id/permissions", roleHandler.AssignPermissions)
	}
}

func (r *Router) registerPermissionRoutes(authorized *gin.RouterGroup, deps *RouteDeps) {
	permHandler := deps.PermissionHandler

	// 例外：无需 RBAC（前端菜单渲染需要权限树）
	authorized.GET("/permissions/tree", permHandler.Tree)

	// 需要 RBAC 的管理路由
	permMgmt := authorized.Group("/permissions")
	permMgmt.Use(r.RBAC())
	{
		permMgmt.POST("", permHandler.Create)
		permMgmt.PUT("/:id", permHandler.Update)
		permMgmt.DELETE("/:id", permHandler.Delete)
	}
}

func (r *Router) registerFileRoutes(authorized *gin.RouterGroup, deps *RouteDeps) {
	fileHandler := deps.FileHandler
	files := authorized.Group("/files")
	files.Use(r.RBAC())
	{
		files.POST("/upload/init", fileHandler.InitUpload)
		files.POST("/upload/:upload_id/chunk", fileHandler.UploadChunk)
		files.POST("/upload/:upload_id/complete", fileHandler.CompleteUpload)
		files.GET("/upload/:upload_id/progress", fileHandler.UploadProgress)
		files.POST("/upload/check", fileHandler.CheckFile)
		files.GET("", fileHandler.List)
		files.GET("/export", fileHandler.ExportCSV)
		files.POST("/batch-delete", fileHandler.BatchDelete)
		files.GET("/:id", fileHandler.GetByID)
		files.GET("/:id/download", fileHandler.Download)
		files.DELETE("/:id", fileHandler.Delete)
	}
}

func (r *Router) registerAlgoPackageRoutes(authorized *gin.RouterGroup, deps *RouteDeps) {
	algoHandler := deps.AlgorithmPackageHandler
	algos := authorized.Group("/algorithmpackages")
	algos.Use(r.RBAC())
	{
		algos.GET("", algoHandler.List)
		algos.POST("/upload", middleware.UploadProtection(r.config.MaxUploadConcurrency), algoHandler.UploadAlgorithm)
		algos.GET("/:id", algoHandler.GetByID)
		algos.DELETE("/:id", algoHandler.Delete)
	}
}

func (r *Router) registerAuditRoutes(authorized *gin.RouterGroup, deps *RouteDeps) {
	auditHandler := deps.AuditHandler
	auditLogs := authorized.Group("/audit-logs")
	auditLogs.Use(r.RBAC())
	{
		auditLogs.GET("", auditHandler.List)
		auditLogs.GET("/export", auditHandler.ExportCSV)
	}
}

func (r *Router) registerGeneralTaskRoutes(authorized *gin.RouterGroup, deps *RouteDeps) {
	taskHandler := deps.TaskHandler
	tasks := authorized.Group("/tasks")
	tasks.Use(r.RBAC())
	{
		tasks.POST("", taskHandler.Create)
		tasks.GET("", taskHandler.List)
		tasks.GET("/export", taskHandler.ExportCSV)
		tasks.POST("/batch-cancel", taskHandler.BatchCancel)
		tasks.GET("/:id", taskHandler.GetByID)
		tasks.POST("/:id/cancel", taskHandler.Cancel)
	}
}

func (r *Router) registerSystemRoutes(authorized *gin.RouterGroup, v1 *gin.RouterGroup, deps *RouteDeps) {
	brandHandler := deps.BrandHandler
	// Brand config GET: 无需登录也不需要 RBAC（前端品牌展示）
	v1.GET("/system/brand-config", brandHandler.GetConfig)

	// Brand config mutate + Mail config + License + Dashboard: all need RBAC
	sysRBAC := authorized.Group("")
	sysRBAC.Use(r.RBAC())
	{
		brandCfg := sysRBAC.Group("/system/brand-config")
		brandCfg.PUT("", brandHandler.SaveConfig)
		brandCfg.POST("/logo", brandHandler.UploadLogo)

		mailCfg := sysRBAC.Group("/system/mail-config")
		mailCfg.GET("", deps.MailHandler.GetConfig)
		mailCfg.PUT("", deps.MailHandler.SaveConfig)
		mailCfg.POST("/test-smtp", deps.MailHandler.TestSMTP)
		mailCfg.POST("/test-imap", deps.MailHandler.TestIMAP)
		mailCfg.POST("/sync-imap", deps.MailHandler.SyncIMAP)

		if deps.LicenseHandler != nil {
			license := sysRBAC.Group("/license")
			license.GET("/fingerprint", deps.LicenseHandler.GetFingerprint)
			license.POST("/upload", deps.LicenseHandler.Upload)
			license.GET("/active", deps.LicenseHandler.GetActive)
			license.GET("/check", deps.LicenseHandler.CheckAuth)
			license.GET("", deps.LicenseHandler.List)
		}

		sysRBAC.GET("/dashboard/stats", deps.DashboardHandler.Stats)
	}

	// System Management (no RBAC, relies on internal auth)
	if deps.SystemHandler != nil {
		systemRouter := NewSystemRouter(deps.SystemHandler, r.jwtManager)
		systemRouter.RegisterRoutes(authorized)
	}
}

func (r *Router) registerFeedbackRoutes(authorized *gin.RouterGroup, deps *RouteDeps) {
	feedbackHandler := deps.FeedbackHandler

	// 例外：任意认证用户可提交反馈，无需 RBAC
	authorized.POST("/feedback", feedbackHandler.Create)

	// 需要 RBAC 的管理路由
	feedbackMgmt := authorized.Group("/feedback")
	feedbackMgmt.Use(r.RBAC())
	{
		feedbackMgmt.GET("", feedbackHandler.List)
		feedbackMgmt.GET("/export", feedbackHandler.ExportCSV)
		feedbackMgmt.PUT("/batch-status", feedbackHandler.BatchUpdateStatus)
		feedbackMgmt.PUT("/:id/status", feedbackHandler.UpdateStatus)
	}
}

func (r *Router) registerDeviceRoutes(authorized *gin.RouterGroup, deps *RouteDeps) {
	deviceHandler := deps.DeviceHandler
	devices := authorized.Group("/devices")
	devices.Use(r.RBAC())
	{
		devices.GET("", deviceHandler.List)
		devices.POST("", deviceHandler.Create)
		devices.GET("/export", deviceHandler.ExportCSV)
		devices.POST("/import", deviceHandler.ImportCSV)
		devices.POST("/batch-delete", deviceHandler.BatchDelete)
		devices.GET("/:id", deviceHandler.GetByID)
		devices.PUT("/:id", deviceHandler.Update)
		devices.DELETE("/:id", deviceHandler.Delete)
		devices.POST("/:id/test", deviceHandler.TestConnection)
	}

	deviceGroupHandler := deps.DeviceGroupHandler
	deviceGroups := authorized.Group("/device-groups")
	deviceGroups.Use(r.RBAC())
	{
		deviceGroups.GET("", deviceGroupHandler.List)
		deviceGroups.POST("", deviceGroupHandler.Create)
		deviceGroups.GET("/:id", deviceGroupHandler.GetByID)
		deviceGroups.PUT("/:id", deviceGroupHandler.Update)
		deviceGroups.DELETE("/:id", deviceGroupHandler.Delete)
	}
}

func (r *Router) registerMediaRoutes(authorized *gin.RouterGroup, v1 *gin.RouterGroup, deps *RouteDeps) {
	// Media streaming (ZLM webhooks - internal, no auth)
	mediaWebhookHandler, mediaPlayHandler, mediaRecordingHandler, deviceStagingHandler := provideMediaServices(r.db, r.config, deps.StreamManager, deps.SIPService)
	// Register ZLM webhooks at root level with secret validation
	zlmGroup := r.engine.Group("")
	zlmGroup.Use(middleware.ZLMWebhookAuth(r.config.ZLMSecret))
	mediaWebhookHandler.RegisterRoutes(zlmGroup)

	// Device Staging (Discover results)
	staging := authorized.Group("/device-staging")
	staging.Use(r.RBAC())
	{
		staging.GET("", deviceStagingHandler.List)
		staging.POST("/batch-import", deviceStagingHandler.BatchImport)
		staging.POST("/batch-ignore", deviceStagingHandler.BatchIgnore)
		staging.POST("/scan-onvif", deviceStagingHandler.ScanONVIF)
		staging.POST("/:id/import", deviceStagingHandler.Import)
		staging.POST("/:id/ignore", deviceStagingHandler.Ignore)
	}

	// Media playback API (under /api/v1/media, jwt auth only, no RBAC)
	mediaPlayGroup := v1.Group("/media")
	mediaPlayGroup.Use(middleware.Auth(r.jwtManager))
	mediaPlayHandler.RegisterRoutes(mediaPlayGroup)
	mediaRecordingHandler.RegisterRoutes(mediaPlayGroup)

	// GB28181 设备管理路由
	if deps.GB28181Handler != nil {
		gb28181 := authorized.Group("/gb28181")
		gb28181.Use(r.RBAC())
		{
			gb28181.GET("/devices", deps.GB28181Handler.List)
			gb28181.POST("/devices", deps.GB28181Handler.Create)
			gb28181.POST("/devices/batch-delete", deps.GB28181Handler.BatchDelete)
			gb28181.GET("/devices/:id", deps.GB28181Handler.GetByID)
			gb28181.PUT("/devices/:id", deps.GB28181Handler.Update)
			gb28181.DELETE("/devices/:id", deps.GB28181Handler.Delete)
			gb28181.POST("/devices/:id/catalog", deps.GB28181Handler.TriggerCatalog)
			gb28181.GET("/devices/:id/channels", deps.GB28181Handler.GetChannels)
			gb28181.GET("/catalog-tasks/:task_id", deps.GB28181Handler.GetCatalogTaskStatus)
			gb28181.GET("/nvrs", deps.GB28181Handler.ListNVRs)
			gb28181.GET("/nvrs/:id/channels", deps.GB28181Handler.GetNVRChannels)
		}
	}

	// GB28181 媒体路由
	if deps.MediaGB28181Handler != nil {
		mediaGB28181 := authorized.Group("/media/gb28181")
		mediaGB28181.Use(r.RBAC())
		{
			mediaGB28181.POST("/live/start", deps.MediaGB28181Handler.StartLive)
			mediaGB28181.POST("/live/stop", deps.MediaGB28181Handler.StopLive)
			mediaGB28181.POST("/playback/start", deps.MediaGB28181Handler.StartPlayback)
			mediaGB28181.POST("/playback/control", deps.MediaGB28181Handler.PlaybackControl)
			mediaGB28181.POST("/playback/stop", deps.MediaGB28181Handler.StopPlayback)
		}
	}

	// GB28181 配置路由
	if deps.GB28181ConfigHandler != nil {
		gbConfig := authorized.Group("/system/gb28181")
		gbConfig.Use(r.RBAC())
		{
			gbConfig.GET("/config", deps.GB28181ConfigHandler.GetConfig)
			gbConfig.PUT("/config", deps.GB28181ConfigHandler.UpdateConfig)
		}
	}
}

func (r *Router) registerPersonRoutes(authorized *gin.RouterGroup, deps *RouteDeps) {
	personHandler := deps.PersonHandler

	// 例外：图片预览无需 RBAC（URL 包含签名 token）
	personsImage := authorized.Group("/persons/image")
	{
		personsImage.GET("/:filename", personHandler.ViewImage)
	}

	// 需要 RBAC 的 person CRUD
	persons := authorized.Group("/persons")
	persons.Use(r.RBAC())
	{
		persons.GET("", personHandler.List)
		persons.POST("", personHandler.Create)
		persons.GET("/export", personHandler.ExportExcel)
		persons.POST("/search-by-face", personHandler.SearchByFace)
		persons.POST("/batch-delete", personHandler.BatchDelete)
		persons.POST("/batch-toggle", personHandler.BatchToggle)
		persons.POST("/batch-retry-embedding", personHandler.BatchRetryEmbedding)
		persons.GET("/:id", personHandler.GetByID)
		persons.PUT("/:id", personHandler.Update)
		persons.DELETE("/:id", personHandler.Delete)
		persons.POST("/:id/retry-embedding", personHandler.RetryEmbedding)
	}

	personGroups := authorized.Group("/person-groups")
	personGroups.Use(r.RBAC())
	{
		personGroups.GET("", personHandler.ListGroups)
		personGroups.POST("", personHandler.CreateGroup)
		personGroups.PUT("/:id", personHandler.UpdateGroup)
		personGroups.DELETE("/:id", personHandler.DeleteGroup)
	}

	personTags := authorized.Group("/person-tags")
	personTags.Use(r.RBAC())
	{
		personTags.GET("", personHandler.ListTags)
		personTags.POST("", personHandler.CreateTag)
		personTags.PUT("/:id", personHandler.UpdateTag)
		personTags.DELETE("/:id", personHandler.DeleteTag)
	}

	personImports := authorized.Group("/person-import-tasks")
	personImports.Use(r.RBAC())
	{
		personImports.GET("", personHandler.ListImportTasks)
		personImports.POST("", personHandler.Import)
		personImports.POST("/by-url", personHandler.ImportByURL)
		personImports.GET("/:id", personHandler.GetImportTask)
	}
}

func (r *Router) registerSmartRecordRoutes(authorized *gin.RouterGroup, deps *RouteDeps) {
	smartRecordHandler := deps.SmartRecordHandler

	smartRecords := authorized.Group("/smart-records")
	smartRecords.Use(r.RBAC())
	{
		smartRecords.GET("/category-codes", smartRecordHandler.ListCategoryCodes)
		smartRecords.GET("", smartRecordHandler.List)
		smartRecords.GET("/export", smartRecordHandler.ExportCSV)
		smartRecords.POST("/batch-delete", smartRecordHandler.BatchDelete)
		smartRecords.PUT("/:id/alarm-status", smartRecordHandler.UpdateAlarmStatus)
		smartRecords.POST("/export-selected", smartRecordHandler.ExportSelectedCSV)
	}
}

// NewAsynqServer creates an Asynq server for task processing.
func NewAsynqServer(rdb *redis.Client) *asynq.Server {
	return task.NewServer(rdb)
}

// NewAsynqScheduler creates an Asynq scheduler for periodic task scheduling.
func NewAsynqScheduler(rdb *redis.Client) *asynq.Scheduler {
	return task.NewScheduler(rdb)
}

// NewAsynqMux creates an Asynq mux with all task handlers registered.
func NewAsynqMux(db *gorm.DB, rdb *redis.Client, cfg *Config, mqttClient mqtt.Client, syncManager *mqttsync.MqttSyncManager, scheduler *asynq.Scheduler, hub *ws.Hub) *asynq.ServeMux {
	deviceRepo := repository.NewDeviceRepository(db)
	zlmClient := provideZLMClient(cfg)

	storageSvc := service.NewStorageService(db, rdb)
	cronCleanupHandler := task.NewCronCleanupHandler(storageSvc)
	thresholdCleanupHandler := task.NewThresholdCleanupHandler(storageSvc)

	// Since we are wiring up AIVisionTaskService manually here for the background worker:
	aiTaskRepo := repository.NewAIVisionTaskRepository(db)
	aiScheduleRepo := repository.NewAITimeScheduleRepository(db)
	algorithmPackageRepo := repository.NewAlgorithmPackageRepository(db)
	nodeRepo := repository.NewEdgeNodeRepository(db)
	nodeAlgoRepo := repository.NewEdgeNodeAlgorithmRepository(db)
	gbDeviceRepo := repository.NewGB28181DeviceRepository(db)
	mediaStreamRepo := repository.NewMediaStreamRepository(db)
	engineClient := provideEngineClient(mqttClient, syncManager)
	streamManager := provideStreamManager(engineClient, deviceRepo, mediaStreamRepo)

	deviceStatusHandler := task.NewDeviceStatusHandler(deviceRepo, zlmClient, streamManager)
	// 由于 Asynq worker 自身消费任务队列，这里传入 nil taskClient 避免循环依赖（worker 内的 SIPService 不需要再派发任务）。
	smartRecordRepo := repository.NewSmartRecordRepository(db)
	deviceSipConfigRepo := repository.NewDeviceSipConfigRepository(db)
	deviceRepoV2 := repository.NewDeviceRepositoryV2(db)
	auditRepo := repository.NewAuditRepository(db)
	streamSessionRepo := repository.NewGB28181StreamSessionRepository(db)
	sipSvc := provideSIPServiceWithZLM(deviceRepo, gbDeviceRepo, mediaStreamRepo, smartRecordRepo, deviceSipConfigRepo, deviceRepoV2, nil, zlmClient, streamManager, cfg, nil, nil, auditRepo, streamSessionRepo)
	policy := service.NewInferenceNodePolicy(nodeRepo, nodeAlgoRepo)
	aiTaskSvc := service.NewAIVisionTaskService(aiTaskRepo, aiScheduleRepo, algorithmPackageRepo, deviceRepo, sipSvc, streamManager, policy)

	mux := task.NewMux(provideMailServiceForAsynq(db), deviceStatusHandler, cronCleanupHandler, thresholdCleanupHandler, aiTaskSvc)

	// Phase 3: Edge Node Task Scheduler (periodic evaluation of cron/one-shot tasks)
	edgeNodeScheduledTaskRepo := repository.NewEdgeNodeScheduledTaskRepository(db)
	edgeNodeTaskExecutionRepo := repository.NewEdgeNodeTaskExecutionRepository(db)
	edgeNodeScheduledTaskSvc := service.NewEdgeNodeScheduledTaskService(
		edgeNodeScheduledTaskRepo,
		edgeNodeTaskExecutionRepo,
		nodeRepo, // reuse from above
		mqttClient,
		syncManager,
	)
	edgeNodeTerminalSvc := service.NewEdgeNodeTerminalService(mqttClient)
	edgeNodeTaskSchedulerHandler := task.NewEdgeNodeTaskSchedulerHandler(edgeNodeScheduledTaskSvc, edgeNodeTerminalSvc)
	edgeNodeTaskSchedulerHandler.RegisterHandlers(mux)
	if scheduler != nil {
		task.RegisterEdgeNodePeriodicTasks(scheduler)
	}

	// Edge Node Status Checker
	edgeNodeStatusTask := task.NewEdgeNodeStatusTask(nodeRepo, aiTaskRepo, hub, cfg.Engine.HeartbeatTimeoutSec, service.NewHeartbeatStore(rdb))
	edgeNodeStatusTask.RegisterHandlers(mux)
	edgeNodeStatusTask.RegisterPeriodic(scheduler, cfg.Engine.HeartbeatCheckIntervalSec)

	// Edge Node Algorithm Retry Handler (R5: auto-retry with exponential backoff)
	edgeNodeAlgoRetryTask := task.NewEdgeNodeAlgorithmRetryTask(nodeAlgoRepo, nodeRepo)
	edgeNodeAlgoRetryTask.RegisterHandlers(mux)
	if scheduler != nil {
		// Run retry check every 5 minutes (faster than heartbeat interval to catch transient failures)
		scheduler.Register("@every 5m", asynq.NewTask(task.TypeEdgeNodeAlgorithmRetry, nil))
		zap.L().Info("registered periodic edge node algorithm retry check", zap.String("cron", "@every 5m"))
	}

	// Edge Node State Reconciliation Worker
	edgeStateWorker := task.NewEdgeStateWorker(aiTaskRepo, nodeRepo, rdb, engineClient)
	mux.HandleFunc(task.TaskReconcileEdgeState, edgeStateWorker.HandleReconcileEdgeState)

	// Edge Scheduled Task Patrol
	schTaskRepo := repository.NewEdgeScheduledTaskRepository(db)
	schRecordRepo := repository.NewEdgeScheduledTaskRecordRepository(db)
	schTagRepo := repository.NewEdgeNodeTagRepository(db)
	edgeScheduledTaskSvc := service.NewEdgeScheduledTaskService(
		schTaskRepo, schRecordRepo, schTagRepo, nodeRepo, mqttClient, syncManager, hub,
	)
	edgeScheduledTaskHandler := task.NewEdgeScheduledTaskHandler(edgeScheduledTaskSvc)
	edgeScheduledTaskHandler.RegisterHandlers(mux)
	edgeScheduledTaskHandler.RegisterPeriodic(scheduler)

		// Terminal Session Cleanup Task
		termSessionCleanupRepo := repository.NewTerminalSessionRepository(db)
		termSessionCleanupPool := service.NewSSHPool()
		termSessionCleanupSvc := service.NewTerminalSessionService(termSessionCleanupRepo, nodeRepo, termSessionCleanupPool, zap.L())
		termSessionCleanupHandler := task.NewTerminalSessionHandler(termSessionCleanupRepo, termSessionCleanupSvc)
		termSessionCleanupHandler.RegisterHandlers(mux)
		termSessionCleanupHandler.RegisterPeriodic(scheduler)

	// Metrics Retention Handler (daily cleanup of old edge node metrics)
	metricsRepo := repository.NewEdgeNodeMetricsRepository(db)
	metricsSvc := service.NewEdgeNodeMetricsService(metricsRepo, hub)
	metricsRetentionHandler := task.NewMetricsRetentionHandler(metricsSvc)
	metricsRetentionHandler.RegisterHandlers(mux)
	if scheduler != nil {
		// Run retention cleanup once per day at 3:00 AM
		scheduler.Register("0 3 * * *", asynq.NewTask(task.MetricsRetentionTaskType, nil))
		zap.L().Info("registered periodic edge node metrics retention",
			zap.String("cron", "0 3 * * *"))
	}

	// 人员相关任务处理器依赖本地存储作为人脸图片载体。存储初始化失败时记录告警
	// 并跳过注册，避免后续任务运行时再崩溃。
	avatarStorage, err := provideAvatarStorage(cfg)
	if err != nil {
		zap.L().Error("init avatar storage failed, person task handlers skipped", zap.Error(err))
		return mux
	}
	personRepo := repository.NewPersonRepository(db)
	embeddingRepo := repository.NewPersonEmbeddingRepository(db)
	importTaskRepo := repository.NewImportTaskRepository(db)
	faceEmbeddingScheduler := service.NewFaceEmbeddingScheduler(nodeRepo, rdb)
	faceLibrarySyncSvc := service.NewFaceLibrarySyncService(embeddingRepo, algorithmPackageRepo, faceEmbeddingScheduler, engineClient)
	taskClient := task.NewClient(rdb)
	task.NewPersonEmbeddingHandler(personRepo, embeddingRepo, algorithmPackageRepo, faceLibrarySyncSvc, faceEmbeddingScheduler, engineClient, avatarStorage).RegisterHandlers(mux)
	task.NewPersonImportHandler(personRepo, embeddingRepo, importTaskRepo, avatarStorage, taskClient).RegisterHandlers(mux)

	return mux
}
