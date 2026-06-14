package router

import (
	"github.com/gin-gonic/gin"
	"github.com/niko-admin/niko-admin/internal/handler"
	"github.com/niko-admin/niko-admin/internal/pkg/jwt"
)

// SystemRouter 系统管理路由
type SystemRouter struct {
	systemHandler *handler.SystemHandler
	jwtManager    *jwt.Manager
}

// NewSystemRouter 创建系统管理路由
func NewSystemRouter(
	systemHandler *handler.SystemHandler,
	jwtManager *jwt.Manager,
) *SystemRouter {
	return &SystemRouter{
		systemHandler: systemHandler,
		jwtManager:    jwtManager,
	}
}

// RegisterRoutes 注册系统管理路由
func (r *SystemRouter) RegisterRoutes(router *gin.RouterGroup) {
	system := router.Group("/system")
	// 注意：router 是 authorized 组（已有 Auth + Audit），
	// 所有子路由自动继承，无需重复挂载 Auth 中间件

	// 管理员可配置的系统信息（设备型号、部署位置、描述）
	info := system.Group("/info")
	{
		info.GET("", r.systemHandler.GetSystemInfo)
		info.PUT("", r.systemHandler.UpdateSystemInfo)
	}

	// 运行状态 (只读，无需审计)
	status := system.Group("/status")
	{
		status.GET("/realtime", r.systemHandler.GetRealtimeStatus)
		status.GET("/resources", r.systemHandler.GetResourceStatus)
		status.GET("/services", r.systemHandler.GetServiceStatus)
		status.GET("/engine", r.systemHandler.GetEngineStatus)
		status.GET("/streams", r.systemHandler.GetStreamsStatus)
		status.GET("/sip", r.systemHandler.GetSIPStatus)
		status.GET("/history", r.systemHandler.GetStatusHistory)
	}

	// 网络配置 (需要审计)
	network := system.Group("/network")
	{
		network.GET("", r.systemHandler.GetNetworkInterfaces)
		network.POST("/apply", r.systemHandler.ApplyNetworkConfig)
		network.POST("/confirm", r.systemHandler.ConfirmNetworkConfig)
		network.POST("/rollback", r.systemHandler.RollbackNetworkConfig)
	}

	// 时间配置 (需要审计)
	timeConfig := system.Group("/time")
	{
		timeConfig.GET("", r.systemHandler.GetTimeConfig)
		timeConfig.POST("/manual", r.systemHandler.SetManualTime)
		timeConfig.GET("/timezone", r.systemHandler.GetTimezone)
		timeConfig.PUT("/timezone", r.systemHandler.SetTimezone)

		// NTP 配置
		ntp := timeConfig.Group("/ntp")
		{
			ntp.GET("", r.systemHandler.GetNTPConfig)
			ntp.POST("/servers", r.systemHandler.AddNTPServer)
			ntp.DELETE("/servers/:host", r.systemHandler.RemoveNTPServer)
			ntp.POST("/test", r.systemHandler.TestNTPServers)
			ntp.POST("/sync", r.systemHandler.SyncNTP)
			ntp.GET("/recommended", r.systemHandler.GetRecommendedNTPServers)
		}
	}

	// 告警上报 (Webhook)
	webhook := system.Group("/webhook")
	{
		webhook.GET("", r.systemHandler.ListWebhooks)
		webhook.POST("", r.systemHandler.CreateWebhook)
		webhook.GET("/:id", r.systemHandler.GetWebhook)
		webhook.PUT("/:id", r.systemHandler.UpdateWebhook)
		webhook.DELETE("/:id", r.systemHandler.DeleteWebhook)
		webhook.POST("/:id/test", r.systemHandler.TestWebhook)
		webhook.GET("/logs", r.systemHandler.ListWebhookLogs)
	}

	// 存储配置
	storage := system.Group("/storage")
	{
		storage.GET("/config", r.systemHandler.GetStorageConfig)
		storage.PUT("/config", r.systemHandler.UpdateStorageConfig)
		storage.GET("/cleanup-logs", r.systemHandler.ListCleanupLogs)
		storage.POST("/cleanup/run", r.systemHandler.RunCleanup)
	}
}
