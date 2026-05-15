// Package router provides HTTP route registration for niko-admin.
package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/gorm"

	"github.com/niko-admin/niko-admin/internal/handler"
	"github.com/niko-admin/niko-admin/internal/middleware"
	"github.com/niko-admin/niko-admin/internal/pkg/jwt"
	"github.com/niko-admin/niko-admin/internal/pkg/ws"
	"github.com/niko-admin/niko-admin/internal/task"
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
	AppEnv            string
	AllowOrigins      []string
	RequestsPerMinute int
}

// New creates a new Router with all dependencies wired.
func New(db *gorm.DB, rdb *redis.Client, jwtManager *jwt.Manager, hub *ws.Hub, cfg *Config) *Router {
	engine := gin.New()

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
}

func (r *Router) setupRoutes() {
	v1 := r.engine.Group("/api/v1")

	// Auth (no auth required)
	authHandler := handler.NewAuthHandler(r.db, r.rdb, r.jwtManager)
	v1.POST("/auth/login", authHandler.Login)
	v1.POST("/auth/refresh", authHandler.Refresh)
	v1.POST("/auth/logout", middleware.Auth(r.jwtManager), authHandler.Logout)
	v1.GET("/auth/me", middleware.Auth(r.jwtManager), authHandler.Me)
	v1.PUT("/auth/password", middleware.Auth(r.jwtManager), authHandler.ChangePassword)

	// WebSocket
	wsHandler := handler.NewWSHandler(r.hub, r.jwtManager)
	r.engine.GET("/api/v1/ws", wsHandler.HandleWebSocket)

	// Protected routes
	authorized := v1.Group("")
	authorized.Use(middleware.Auth(r.jwtManager))

	// Users
	userHandler := handler.NewUserHandler(r.db)
	users := authorized.Group("/users")
	{
		users.GET("", userHandler.List)
		users.POST("", middleware.RBAC(r.rdb, r.db), userHandler.Create)
		users.GET("/:id", userHandler.GetByID)
		users.PUT("/:id", middleware.RBAC(r.rdb, r.db), userHandler.Update)
		users.DELETE("/:id", middleware.RBAC(r.rdb, r.db), userHandler.Delete)
	}

	// Roles
	roleHandler := handler.NewRoleHandler(r.db, r.rdb)
	roles := authorized.Group("/roles")
	{
		roles.GET("", roleHandler.List)
		roles.POST("", middleware.RBAC(r.rdb, r.db), roleHandler.Create)
		roles.GET("/:id", roleHandler.GetByID)
		roles.PUT("/:id", middleware.RBAC(r.rdb, r.db), roleHandler.Update)
		roles.DELETE("/:id", middleware.RBAC(r.rdb, r.db), roleHandler.Delete)
		roles.GET("/:id/permissions", roleHandler.GetPermissions)
		roles.PUT("/:id/permissions", middleware.RBAC(r.rdb, r.db), roleHandler.AssignPermissions)
	}

	// Permissions
	permissionHandler := handler.NewPermissionHandler(r.db)
	permissions := authorized.Group("/permissions")
	{
		permissions.GET("/tree", permissionHandler.Tree)
		permissions.POST("", middleware.RBAC(r.rdb, r.db), permissionHandler.Create)
	}

	// Files
	fileHandler := handler.NewFileHandler(r.db)
	files := authorized.Group("/files")
	{
		files.POST("/upload/init", fileHandler.InitUpload)
		files.POST("/upload/:upload_id/chunk", fileHandler.UploadChunk)
		files.POST("/upload/:upload_id/complete", fileHandler.CompleteUpload)
		files.GET("/upload/:upload_id/progress", fileHandler.UploadProgress)
		files.POST("/upload/check", fileHandler.CheckFile)
		files.GET("", fileHandler.List)
		files.GET("/:id", fileHandler.GetByID)
		files.GET("/:id/download", fileHandler.Download)
		files.DELETE("/:id", fileHandler.Delete)
	}

	// Audit Logs
	auditHandler := handler.NewAuditHandler(r.db)
	authorized.GET("/audit-logs", middleware.RBAC(r.rdb, r.db), auditHandler.List)

	// Tasks
	taskHandler := handler.NewTaskHandler(r.db)
	tasks := authorized.Group("/tasks")
	{
		tasks.POST("", taskHandler.Create)
		tasks.GET("", taskHandler.List)
		tasks.GET("/:id", taskHandler.GetByID)
		tasks.POST("/:id/cancel", taskHandler.Cancel)
	}

	// Dashboard
	dashboardHandler := handler.NewDashboardHandler(r.db)
	authorized.GET("/dashboard/stats", dashboardHandler.Stats)

	// Swagger UI (non-production only)
	if r.config.AppEnv != "prod" {
		r.engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	// Health check
	r.engine.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
}

// NewAsynqServer creates an Asynq server for task processing.
func NewAsynqServer(rdb *redis.Client) *asynq.Server {
	return task.NewServer(rdb)
}

// NewAsynqMux creates an Asynq mux with all task handlers registered.
func NewAsynqMux() *asynq.ServeMux {
	return task.NewMux()
}
