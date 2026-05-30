// Package router provides HTTP route registration for niko-admin.
package router

import (
	"net/http"
	"strings"
	"time"

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
	"github.com/niko-admin/niko-admin/internal/pkg/response"
	"github.com/niko-admin/niko-admin/internal/pkg/ws"
	"github.com/niko-admin/niko-admin/internal/repository"
	"github.com/niko-admin/niko-admin/internal/service"
	"github.com/niko-admin/niko-admin/internal/task"
	"github.com/niko-admin/niko-admin/pkg/storage"
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
}

// New creates a new Router with all dependencies wired.
func New(db *gorm.DB, rdb *redis.Client, jwtManager *jwt.Manager, hub *ws.Hub, cfg *Config, accessLogger *zap.Logger) *Router {
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

	// Create repositories
	userRepo := repository.NewUserRepository(r.db)
	roleRepo := repository.NewRoleRepository(r.db)
	permRepo := repository.NewPermissionRepository(r.db)
	auditRepo := repository.NewAuditRepository(r.db)
	fileRepo := repository.NewFileRepository(r.db)
	taskRepo := repository.NewTaskRepository(r.db)
	dashRepo := repository.NewDashboardRepository(r.db)
	brandConfigRepo := repository.NewBrandConfigRepository(r.db)
	mailConfigRepo := repository.NewMailConfigRepository(r.db)
	emailTokenRepo := repository.NewEmailTokenRepository(r.db)
	inboundEmailRepo := repository.NewInboundEmailRepository(r.db)
	feedbackRepo := repository.NewFeedbackRepository(r.db)

	// Create services
	avatarStorage, err := storage.NewLocalStorage("uploads", "/uploads")
	if err != nil {
		zap.L().Fatal("create avatar storage failed", zap.Error(err))
	}
	auditSvc := service.NewAuditService(auditRepo)
	authSvc := service.NewAuthService(userRepo, permRepo, r.rdb, r.jwtManager, avatarStorage)
	userSvc := service.NewUserService(userRepo)
	roleSvc := service.NewRoleService(roleRepo, userRepo, r.rdb)
	var permCache cache.Cache
	if r.config.PermissionTreeRedisEnable && r.rdb != nil {
		permCache = cache.NewRedisCache(r.rdb)
	} else {
		if r.config.PermissionTreeRedisEnable && r.rdb == nil {
			zap.L().Warn("permission tree redis cache enabled but redis client is nil, fallback to memory")
		}
		permCache = cache.NewMemoryCache(5 * time.Minute)
	}
	permSvc := service.NewPermissionService(permRepo, permCache)
	rbacCache := permCache
	fileSvc := service.NewFileService(fileRepo, service.FileOptions{
		MaxFileSizeBytes:  int64(r.config.MaxFileSizeMB) << 20,
		MaxChunkSizeBytes: int64(r.config.ChunkSizeMB) << 20,
	})
	taskSvc := service.NewTaskService(taskRepo)
	dashSvc := service.NewDashboardService(dashRepo)
	brandSvc := service.NewBrandServiceWithStorage(brandConfigRepo, avatarStorage)
	mailSvc := service.NewMailService(mailConfigRepo, inboundEmailRepo, feedbackRepo)
	emailVerificationSvc := service.NewEmailVerificationService(emailTokenRepo, userRepo, mailSvc)
	feedbackSvc := service.NewFeedbackService(feedbackRepo, mailSvc)

	// Auth (no auth required)
	authHandler := handler.NewAuthHandlerWithEmail(authSvc, emailVerificationSvc, auditSvc)
	v1.POST("/auth/login", authHandler.Login)
	v1.POST("/auth/password-reset/request", authHandler.RequestPasswordReset)
	v1.POST("/auth/password-reset/confirm", authHandler.ResetPassword)
	v1.POST("/auth/refresh", authHandler.Refresh)
	v1.POST("/auth/logout", middleware.Auth(r.jwtManager), middleware.Audit(auditSvc), authHandler.Logout)
	v1.GET("/auth/me", middleware.Auth(r.jwtManager), authHandler.Me)
	v1.PUT("/auth/password", middleware.Auth(r.jwtManager), authHandler.ChangePassword)
	v1.PUT("/auth/profile", middleware.Auth(r.jwtManager), authHandler.UpdateProfile)
	v1.POST("/auth/avatar", middleware.Auth(r.jwtManager), authHandler.UploadAvatar)

	// WebSocket
	wsHandler := handler.NewWSHandler(r.hub, r.jwtManager, r.config.AllowOrigins)
	r.engine.GET("/api/v1/ws", wsHandler.HandleWebSocket)

	// Protected routes
	authorized := v1.Group("")
	authorized.Use(middleware.Auth(r.jwtManager))
	authorized.Use(middleware.Audit(auditSvc))

	// Users
	userHandler := handler.NewUserHandler(userSvc, authSvc)
	users := authorized.Group("/users")
	{
		users.GET("", middleware.RBAC(rbacCache, r.db), userHandler.List)
		users.POST("", middleware.RBAC(rbacCache, r.db), userHandler.Create)
		users.GET("/:id", middleware.RBAC(rbacCache, r.db), userHandler.GetByID)
		users.PUT("/:id", middleware.RBAC(rbacCache, r.db), userHandler.Update)
		users.DELETE("/:id", middleware.RBAC(rbacCache, r.db), userHandler.Delete)
		users.PUT("/:id/password", middleware.RBAC(rbacCache, r.db), userHandler.ResetPassword)
		users.POST("/:id/avatar", middleware.RBAC(rbacCache, r.db), userHandler.UploadAvatar)
	}

	// Roles
	roleHandler := handler.NewRoleHandler(roleSvc)
	roles := authorized.Group("/roles")
	{
		roles.GET("", roleHandler.List)
		roles.POST("", middleware.RBAC(rbacCache, r.db), roleHandler.Create)
		roles.GET("/:id", middleware.RBAC(rbacCache, r.db), roleHandler.GetByID)
		roles.PUT("/:id", middleware.RBAC(rbacCache, r.db), roleHandler.Update)
		roles.DELETE("/:id", middleware.RBAC(rbacCache, r.db), roleHandler.Delete)
		roles.GET("/:id/permissions", middleware.RBAC(rbacCache, r.db), roleHandler.GetPermissions)
		roles.PUT("/:id/permissions", middleware.RBAC(rbacCache, r.db), roleHandler.AssignPermissions)
	}

	// Permissions
	permHandler := handler.NewPermissionHandler(permSvc)
	permissions := authorized.Group("/permissions")
	{
		permissions.GET("/tree", permHandler.Tree)
		permissions.POST("", middleware.RBAC(rbacCache, r.db), permHandler.Create)
		permissions.PUT("/:id", middleware.RBAC(rbacCache, r.db), permHandler.Update)
		permissions.DELETE("/:id", middleware.RBAC(rbacCache, r.db), permHandler.Delete)
	}

	// Files
	fileHandler := handler.NewFileHandler(fileSvc)
	files := authorized.Group("/files")
	{
		files.POST("/upload/init", middleware.RBAC(rbacCache, r.db), fileHandler.InitUpload)
		files.POST("/upload/:upload_id/chunk", middleware.RBAC(rbacCache, r.db), fileHandler.UploadChunk)
		files.POST("/upload/:upload_id/complete", middleware.RBAC(rbacCache, r.db), fileHandler.CompleteUpload)
		files.GET("/upload/:upload_id/progress", middleware.RBAC(rbacCache, r.db), fileHandler.UploadProgress)
		files.POST("/upload/check", middleware.RBAC(rbacCache, r.db), fileHandler.CheckFile)
		files.GET("", middleware.RBAC(rbacCache, r.db), fileHandler.List)
		files.GET("/:id", middleware.RBAC(rbacCache, r.db), fileHandler.GetByID)
		files.GET("/:id/download", middleware.RBAC(rbacCache, r.db), fileHandler.Download)
		files.DELETE("/:id", middleware.RBAC(rbacCache, r.db), fileHandler.Delete)
	}

	// Audit Logs
	auditHandler := handler.NewAuditHandler(auditSvc)
	authorized.GET("/audit-logs", middleware.RBAC(rbacCache, r.db), auditHandler.List)

	// Tasks
	taskHandler := handler.NewTaskHandler(taskSvc)
	tasks := authorized.Group("/tasks")
	{
		tasks.POST("", middleware.RBAC(rbacCache, r.db), taskHandler.Create)
		tasks.GET("", middleware.RBAC(rbacCache, r.db), taskHandler.List)
		tasks.GET("/:id", middleware.RBAC(rbacCache, r.db), taskHandler.GetByID)
		tasks.POST("/:id/cancel", middleware.RBAC(rbacCache, r.db), taskHandler.Cancel)
	}

	// System brand configuration
	brandHandler := handler.NewBrandHandler(brandSvc)
	v1.GET("/system/brand-config", brandHandler.GetConfig)
	brandConfig := authorized.Group("/system/brand-config")
	{
		brandConfig.PUT("", middleware.RBAC(rbacCache, r.db), brandHandler.SaveConfig)
		brandConfig.POST("/logo", middleware.RBAC(rbacCache, r.db), brandHandler.UploadLogo)
	}

	// System mail configuration
	mailHandler := handler.NewMailHandler(mailSvc)
	mailConfig := authorized.Group("/system/mail-config")
	{
		mailConfig.GET("", middleware.RBAC(rbacCache, r.db), mailHandler.GetConfig)
		mailConfig.PUT("", middleware.RBAC(rbacCache, r.db), mailHandler.SaveConfig)
		mailConfig.POST("/test-smtp", middleware.RBAC(rbacCache, r.db), mailHandler.TestSMTP)
		mailConfig.POST("/test-imap", middleware.RBAC(rbacCache, r.db), mailHandler.TestIMAP)
		mailConfig.POST("/sync-imap", middleware.RBAC(rbacCache, r.db), mailHandler.SyncIMAP)
	}

	// Feedback
	feedbackHandler := handler.NewFeedbackHandler(feedbackSvc)
	feedback := authorized.Group("/feedback")
	{
		feedback.POST("", feedbackHandler.Create)
		feedback.GET("", middleware.RBAC(rbacCache, r.db), feedbackHandler.List)
		feedback.PUT("/:id/status", middleware.RBAC(rbacCache, r.db), feedbackHandler.UpdateStatus)
	}

	// Dashboard
	dashboardHandler := handler.NewDashboardHandler(dashSvc)
	authorized.GET("/dashboard/stats", middleware.RBAC(rbacCache, r.db), dashboardHandler.Stats)

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

// NewAsynqMux creates an Asynq mux with all task handlers registered.
func NewAsynqMux(db *gorm.DB) *asynq.ServeMux {
	mailConfigRepo := repository.NewMailConfigRepository(db)
	inboundEmailRepo := repository.NewInboundEmailRepository(db)
	feedbackRepo := repository.NewFeedbackRepository(db)
	mailSvc := service.NewMailService(mailConfigRepo, inboundEmailRepo, feedbackRepo)
	return task.NewMux(mailSvc)
}
