package router

import (
	"github.com/gin-gonic/gin"

	"github.com/niko-admin/niko-admin/internal/handler"
)

// RegisterAITimeScheduleRoutes 注册 AITimeSchedule CRUD 路由。
// api 必须是已挂载 Auth + Audit 中间件的路由组。
func RegisterAITimeScheduleRoutes(api *gin.RouterGroup, h *handler.AITimeScheduleHandler, rbacMiddleware gin.HandlerFunc) {
	group := api.Group("/ai-time-schedules")
	if rbacMiddleware != nil {
		group.Use(rbacMiddleware)
	}
	{
		group.GET("", h.List)
		group.GET("/all", h.ListAll)
		group.POST("", h.Create)
		group.GET("/:id", h.GetByID)
		group.PUT("/:id", h.Update)
		group.DELETE("/:id", h.Delete)
	}
}
