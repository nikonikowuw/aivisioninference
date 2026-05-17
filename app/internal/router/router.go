// Package router provides HTTP route registration for niko-admin.
package router

import (
	"net/http"
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
	"github.com/niko-admin/niko-admin/internal/pkg/httpx"
	"github.com/niko-admin/niko-admin/internal/pkg/jwt"
	"github.com/niko-admin/niko-admin/internal/pkg/ws"
	"github.com/niko-admin/niko-admin/internal/repository"
	"github.com/niko-admin/niko-admin/internal/service"
	"github.com/niko-admin/niko-admin/internal/task"
	"github.com/niko-admin/niko-admin/pkg/storage"
)

// Router holds all dependencies for route registration.
type Router struct {
	engine     *gin.Engine
	db         *gorm.DB
	rdb        *redis.Client
	jwtManager *jwt.Manager
	hub        *ws.Hub
	config     *Config
}

// Config holds router-level configuration.
type Config struct {
	AppEnv                    string
	AllowOrigins              []string
	RequestsPerMinute         int
	TrustedProxies            []string
	PermissionTreeRedisEnable bool
}

// New creates a new Router with all dependencies wired.
func New(db *gorm.DB, rdb *redis.Client, jwtManager *jwt.Manager, hub *ws.Hub, cfg *Config) *Router {
	engine := gin.New()

	httpx.TrustedProxies = cfg.TrustedProxies

	r := &Router{
		engine:     engine,
		db:         db,
		rdb:        rdb,
		jwtManager: jwtManager,
		hub:        hub,
		config:     cfg,
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
	r.engine.Use(middleware.Logger())
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
	fileSvc := service.NewFileService(fileRepo)
	taskSvc := service.NewTaskService(taskRepo)
	dashSvc := service.NewDashboardService(dashRepo)

	// Auth (no auth required)
	authHandler := handler.NewAuthHandler(authSvc, auditSvc)
	v1.POST("/auth/login", authHandler.Login)
	v1.POST("/auth/refresh", authHandler.Refresh)
	v1.POST("/auth/logout", middleware.Auth(r.jwtManager), middleware.Audit(auditSvc), authHandler.Logout)
	v1.GET("/auth/me", middleware.Auth(r.jwtManager), authHandler.Me)
	v1.PUT("/auth/password", middleware.Auth(r.jwtManager), authHandler.ChangePassword)
	v1.PUT("/auth/profile", middleware.Auth(r.jwtManager), authHandler.UpdateProfile)
	v1.POST("/auth/avatar", middleware.Auth(r.jwtManager), authHandler.UploadAvatar)

	// WebSocket
	wsHandler := handler.NewWSHandler(r.hub, r.jwtManager)
	r.engine.GET("/api/v1/ws", wsHandler.HandleWebSocket)

	// Protected routes
	authorized := v1.Group("")
	authorized.Use(middleware.Auth(r.jwtManager))
	authorized.Use(middleware.Audit(auditSvc))

	// Users
	userHandler := handler.NewUserHandler(userSvc, authSvc)
	users := authorized.Group("/users")
	{
		users.GET("", userHandler.List)
		users.POST("", middleware.RBAC(rbacCache, r.db), userHandler.Create)
		users.GET("/:id", middleware.RBAC(rbacCache, r.db), userHandler.GetByID)
		users.PUT("/:id", middleware.RBAC(rbacCache, r.db), userHandler.Update)
		users.DELETE("/:id", middleware.RBAC(rbacCache, r.db), userHandler.Delete)
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
		files.GET("", fileHandler.List)
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
		tasks.GET("", taskHandler.List)
		tasks.GET("/:id", taskHandler.GetByID)
		tasks.POST("/:id/cancel", middleware.RBAC(rbacCache, r.db), taskHandler.Cancel)
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
	r.engine.Static("/uploads", "uploads")
}

// NewAsynqServer creates an Asynq server for task processing.
func NewAsynqServer(rdb *redis.Client) *asynq.Server {
	return task.NewServer(rdb)
}

// NewAsynqMux creates an Asynq mux with all task handlers registered.
func NewAsynqMux() *asynq.ServeMux {
	return task.NewMux()
}
