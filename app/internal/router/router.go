// Package router 提供 HTTP 路由注册和依赖注入编排，串联 Handler、Service、Repository 各层。
package router

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/niko-admin/niko-admin/internal/middleware"
	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
	"github.com/niko-admin/niko-admin/internal/pkg/httpx"
	"github.com/niko-admin/niko-admin/internal/pkg/jwt"
	"github.com/niko-admin/niko-admin/internal/pkg/response"
	"github.com/niko-admin/niko-admin/internal/pkg/ws"
	"github.com/niko-admin/niko-admin/internal/repository"
	"github.com/niko-admin/niko-admin/internal/service"
	"github.com/niko-admin/niko-admin/internal/task"
)

// Router holds all dependencies for route registration.
type Router struct {
	engine       *gin.Engine
	db           *gorm.DB
	rdb          *redis.Client
	jwtManager   *jwt.Manager
	hub          *ws.Hub
	config       *Config
	accessLogger *zap.Logger
	scheduler    *asynq.Scheduler
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
	LocalUploadDir            string
	LocalPublicURL            string
	ZLMAPIURL                 string
	ZLMSecret                 string
}

// New creates a new Router with all dependencies wired.
func New(db *gorm.DB, rdb *redis.Client, jwtManager *jwt.Manager, hub *ws.Hub, cfg *Config, accessLogger *zap.Logger, scheduler *asynq.Scheduler) *Router {
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

	deps, err := InitializeRouteDeps(r.db, r.rdb, r.jwtManager, r.hub, r.config, r.scheduler)
	if err != nil {
		zap.L().Fatal("initialize route dependencies failed", zap.Error(err))
	}
	rbacCache := deps.RBACCache

	// Auth (no auth required)
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

	// WebSocket
	wsHandler := deps.WSHandler
	r.engine.GET("/api/v1/ws", wsHandler.HandleWebSocket)

	// Protected routes
	authorized := v1.Group("")
	authorized.Use(middleware.Auth(r.jwtManager))
	authorized.Use(middleware.Audit(deps.AuditService))

	// Users
	userHandler := deps.UserHandler
	users := authorized.Group("/users")
	{
		users.GET("", middleware.RBAC(rbacCache, r.db), userHandler.List)
		users.POST("", middleware.RBAC(rbacCache, r.db), userHandler.Create)
		users.GET("/export", middleware.RBAC(rbacCache, r.db), userHandler.ExportCSV)
		users.POST("/import", middleware.RBAC(rbacCache, r.db), userHandler.ImportCSV)
		users.POST("/batch-delete", middleware.RBAC(rbacCache, r.db), userHandler.BatchDelete)
		users.PUT("/batch-status", middleware.RBAC(rbacCache, r.db), userHandler.BatchUpdateStatus)
		users.GET("/:id", middleware.RBAC(rbacCache, r.db), userHandler.GetByID)
		users.PUT("/:id", middleware.RBAC(rbacCache, r.db), userHandler.Update)
		users.DELETE("/:id", middleware.RBAC(rbacCache, r.db), userHandler.Delete)
		users.PUT("/:id/password", middleware.RBAC(rbacCache, r.db), userHandler.ResetPassword)
		users.POST("/:id/avatar", middleware.RBAC(rbacCache, r.db), userHandler.UploadAvatar)
	}

	// Roles
	roleHandler := deps.RoleHandler
	roles := authorized.Group("/roles")
	{
		roles.GET("", roleHandler.List)
		roles.POST("", middleware.RBAC(rbacCache, r.db), roleHandler.Create)
		roles.GET("/export", middleware.RBAC(rbacCache, r.db), roleHandler.ExportCSV)
		roles.POST("/batch-delete", middleware.RBAC(rbacCache, r.db), roleHandler.BatchDelete)
		roles.GET("/:id", middleware.RBAC(rbacCache, r.db), roleHandler.GetByID)
		roles.PUT("/:id", middleware.RBAC(rbacCache, r.db), roleHandler.Update)
		roles.DELETE("/:id", middleware.RBAC(rbacCache, r.db), roleHandler.Delete)
		roles.GET("/:id/permissions", middleware.RBAC(rbacCache, r.db), roleHandler.GetPermissions)
		roles.PUT("/:id/permissions", middleware.RBAC(rbacCache, r.db), roleHandler.AssignPermissions)
	}

	// Permissions
	permHandler := deps.PermissionHandler
	permissions := authorized.Group("/permissions")
	{
		permissions.GET("/tree", permHandler.Tree)
		permissions.POST("", middleware.RBAC(rbacCache, r.db), permHandler.Create)
		permissions.PUT("/:id", middleware.RBAC(rbacCache, r.db), permHandler.Update)
		permissions.DELETE("/:id", middleware.RBAC(rbacCache, r.db), permHandler.Delete)
	}

	// Files
	fileHandler := deps.FileHandler
	files := authorized.Group("/files")
	{
		files.POST("/upload/init", middleware.RBAC(rbacCache, r.db), fileHandler.InitUpload)
		files.POST("/upload/:upload_id/chunk", middleware.RBAC(rbacCache, r.db), fileHandler.UploadChunk)
		files.POST("/upload/:upload_id/complete", middleware.RBAC(rbacCache, r.db), fileHandler.CompleteUpload)
		files.GET("/upload/:upload_id/progress", middleware.RBAC(rbacCache, r.db), fileHandler.UploadProgress)
		files.POST("/upload/check", middleware.RBAC(rbacCache, r.db), fileHandler.CheckFile)
		files.GET("", middleware.RBAC(rbacCache, r.db), fileHandler.List)
		files.GET("/export", middleware.RBAC(rbacCache, r.db), fileHandler.ExportCSV)
		files.POST("/batch-delete", middleware.RBAC(rbacCache, r.db), fileHandler.BatchDelete)
		files.GET("/:id", middleware.RBAC(rbacCache, r.db), fileHandler.GetByID)
		files.GET("/:id/download", middleware.RBAC(rbacCache, r.db), fileHandler.Download)
		files.DELETE("/:id", middleware.RBAC(rbacCache, r.db), fileHandler.Delete)
	}

	// Audit Logs
	auditHandler := deps.AuditHandler
	authorized.GET("/audit-logs", middleware.RBAC(rbacCache, r.db), auditHandler.List)
	authorized.GET("/audit-logs/export", middleware.RBAC(rbacCache, r.db), auditHandler.ExportCSV)

	// Tasks
	taskHandler := deps.TaskHandler
	tasks := authorized.Group("/tasks")
	{
		tasks.POST("", middleware.RBAC(rbacCache, r.db), taskHandler.Create)
		tasks.GET("", middleware.RBAC(rbacCache, r.db), taskHandler.List)
		tasks.GET("/export", middleware.RBAC(rbacCache, r.db), taskHandler.ExportCSV)
		tasks.POST("/batch-cancel", middleware.RBAC(rbacCache, r.db), taskHandler.BatchCancel)
		tasks.GET("/:id", middleware.RBAC(rbacCache, r.db), taskHandler.GetByID)
		tasks.POST("/:id/cancel", middleware.RBAC(rbacCache, r.db), taskHandler.Cancel)
	}

	// System brand configuration
	brandHandler := deps.BrandHandler
	v1.GET("/system/brand-config", brandHandler.GetConfig)
	brandConfig := authorized.Group("/system/brand-config")
	{
		brandConfig.PUT("", middleware.RBAC(rbacCache, r.db), brandHandler.SaveConfig)
		brandConfig.POST("/logo", middleware.RBAC(rbacCache, r.db), brandHandler.UploadLogo)
	}

	// System mail configuration
	mailHandler := deps.MailHandler
	mailConfig := authorized.Group("/system/mail-config")
	{
		mailConfig.GET("", middleware.RBAC(rbacCache, r.db), mailHandler.GetConfig)
		mailConfig.PUT("", middleware.RBAC(rbacCache, r.db), mailHandler.SaveConfig)
		mailConfig.POST("/test-smtp", middleware.RBAC(rbacCache, r.db), mailHandler.TestSMTP)
		mailConfig.POST("/test-imap", middleware.RBAC(rbacCache, r.db), mailHandler.TestIMAP)
		mailConfig.POST("/sync-imap", middleware.RBAC(rbacCache, r.db), mailHandler.SyncIMAP)
	}

	// Feedback
	feedbackHandler := deps.FeedbackHandler
	feedback := authorized.Group("/feedback")
	{
		feedback.POST("", feedbackHandler.Create)
		feedback.GET("", middleware.RBAC(rbacCache, r.db), feedbackHandler.List)
		feedback.GET("/export", middleware.RBAC(rbacCache, r.db), feedbackHandler.ExportCSV)
		feedback.PUT("/batch-status", middleware.RBAC(rbacCache, r.db), feedbackHandler.BatchUpdateStatus)
		feedback.PUT("/:id/status", middleware.RBAC(rbacCache, r.db), feedbackHandler.UpdateStatus)
	}

	// Devices
	deviceHandler := deps.DeviceHandler
	devices := authorized.Group("/devices")
	{
		devices.GET("", middleware.RBAC(rbacCache, r.db), deviceHandler.List)
		devices.POST("", middleware.RBAC(rbacCache, r.db), deviceHandler.Create)
		devices.GET("/export", middleware.RBAC(rbacCache, r.db), deviceHandler.ExportCSV)
		devices.POST("/import", middleware.RBAC(rbacCache, r.db), deviceHandler.ImportCSV)
		devices.POST("/batch-delete", middleware.RBAC(rbacCache, r.db), deviceHandler.BatchDelete)
		devices.GET("/:id", middleware.RBAC(rbacCache, r.db), deviceHandler.GetByID)
		devices.PUT("/:id", middleware.RBAC(rbacCache, r.db), deviceHandler.Update)
		devices.DELETE("/:id", middleware.RBAC(rbacCache, r.db), deviceHandler.Delete)
		devices.POST("/:id/test", middleware.RBAC(rbacCache, r.db), deviceHandler.TestConnection)
	}

	// Device Groups
	deviceGroupHandler := deps.DeviceGroupHandler
	deviceGroups := authorized.Group("/device-groups")
	{
		deviceGroups.GET("", middleware.RBAC(rbacCache, r.db), deviceGroupHandler.List)
		deviceGroups.POST("", middleware.RBAC(rbacCache, r.db), deviceGroupHandler.Create)
		deviceGroups.GET("/:id", middleware.RBAC(rbacCache, r.db), deviceGroupHandler.GetByID)
		deviceGroups.PUT("/:id", middleware.RBAC(rbacCache, r.db), deviceGroupHandler.Update)
		deviceGroups.DELETE("/:id", middleware.RBAC(rbacCache, r.db), deviceGroupHandler.Delete)
	}

	// System Management
	if deps.SystemHandler != nil {
		systemRouter := NewSystemRouter(deps.SystemHandler, r.jwtManager, rbacCache)
		systemRouter.RegisterRoutes(authorized)
	}

	// Dashboard
	dashboardHandler := deps.DashboardHandler
	authorized.GET("/dashboard/stats", middleware.RBAC(rbacCache, r.db), dashboardHandler.Stats)

	// Media streaming (ZLM webhooks - internal, no auth)
	mediaWebhookHandler, mediaPlayHandler, mediaRecordingHandler, deviceStagingHandler := provideMediaServices(r.db, r.config, deps.StreamManager)
	// Register ZLM webhooks at root level with secret validation
	zlmGroup := r.engine.Group("")
	zlmGroup.Use(middleware.ZLMWebhookAuth(r.config.ZLMSecret))
	mediaWebhookHandler.RegisterRoutes(zlmGroup)

	// Device Staging (Discover results)
	staging := authorized.Group("/device-staging")
	{
		staging.GET("", middleware.RBAC(rbacCache, r.db), deviceStagingHandler.List)
		staging.POST("/batch-import", middleware.RBAC(rbacCache, r.db), deviceStagingHandler.BatchImport)
		staging.POST("/batch-ignore", middleware.RBAC(rbacCache, r.db), deviceStagingHandler.BatchIgnore)
		staging.POST("/scan-onvif", middleware.RBAC(rbacCache, r.db), deviceStagingHandler.ScanONVIF)
		staging.POST("/:id/import", middleware.RBAC(rbacCache, r.db), deviceStagingHandler.Import)
		staging.POST("/:id/ignore", middleware.RBAC(rbacCache, r.db), deviceStagingHandler.Ignore)
	}

	// Media playback API (under /api/v1/media)
	mediaPlayGroup := v1.Group("/media")
	mediaPlayGroup.Use(middleware.Auth(r.jwtManager))
	mediaPlayHandler.RegisterRoutes(mediaPlayGroup)
	mediaRecordingHandler.RegisterRoutes(mediaPlayGroup)

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
	uploads.Use(func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("Content-Security-Policy", "default-src 'none'; img-src 'self'; media-src 'self'; style-src 'none'; script-src 'none'")
		c.Next()
	})
	uploads.Static("", "uploads")

	// Frontend static files (SPA)
	r.engine.Static("/assets", "./web/dist/assets")
	r.engine.StaticFile("/favicon.ico", "./web/dist/favicon.ico")
	r.engine.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api") {
			response.Err(c, apperrors.New(apperrors.ErrNotFound, "资源不存在"))
			return
		}
		c.File("./web/dist/index.html")
	})
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
func NewAsynqMux(db *gorm.DB, rdb *redis.Client, cfg *Config) *asynq.ServeMux {
	deviceRepo := repository.NewDeviceRepository(db)
	zlmClient := provideZLMClient(cfg)
	deviceStatusHandler := task.NewDeviceStatusHandler(deviceRepo, zlmClient)

	storageSvc := service.NewStorageService(db, rdb)
	cronCleanupHandler := task.NewCronCleanupHandler(storageSvc)
	thresholdCleanupHandler := task.NewThresholdCleanupHandler(storageSvc)

	return task.NewMux(provideMailServiceForAsynq(db), deviceStatusHandler, cronCleanupHandler, thresholdCleanupHandler)
}
