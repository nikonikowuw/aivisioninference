package router

import (
	"github.com/gin-gonic/gin"

	"github.com/niko-admin/niko-admin/internal/handler"
)

// RegisterAITimeScheduleRoutes 注册 AITimeSchedule CRUD 路由。
func RegisterAITimeScheduleRoutes(api *gin.RouterGroup, h *handler.AITimeScheduleHandler, authMiddleware, rbacMiddleware gin.HandlerFunc) {
	group := api.Group("/ai-time-schedules")
	{
		group.GET("", authMiddleware, rbacMiddleware, h.List)
		group.GET("/all", authMiddleware, rbacMiddleware, h.ListAll)
		group.POST("", authMiddleware, rbacMiddleware, h.Create)
		group.GET("/:id", authMiddleware, rbacMiddleware, h.GetByID)
		group.PUT("/:id", authMiddleware, rbacMiddleware, h.Update)
		group.DELETE("/:id", authMiddleware, rbacMiddleware, h.Delete)
	}
}
