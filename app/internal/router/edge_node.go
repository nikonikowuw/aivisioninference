package router

import (
	"github.com/gin-gonic/gin"
)

func (r *Router) registerEdgeNodeRoutes(authorized *gin.RouterGroup, v1 *gin.RouterGroup, deps *RouteDeps) {
	nodeHandler := deps.EdgeNodeHandler
	nodeMiddleware := deps.EdgeNodeMiddleware

	// Heartbeat route: verified by node-specific JWT token
	v1.POST("/edge-nodes/:id/heartbeat", nodeMiddleware.AuthNode(), nodeHandler.Heartbeat)

	// Admin management routes
	nodes := authorized.Group("/edge-nodes")
	nodes.Use(r.RBAC())
	{
		nodes.GET("", nodeHandler.List)
		nodes.POST("", nodeHandler.Create)
		nodes.GET("/:id", nodeHandler.GetByID)
		nodes.PUT("/:id", nodeHandler.Update)
		nodes.DELETE("/:id", nodeHandler.Delete)
		nodes.POST("/:id/deploy-algo", nodeHandler.DeployAlgorithm)
		nodes.GET("/:id/algorithms", nodeHandler.ListAlgorithms)
		nodes.DELETE("/:id/algorithms/:algo_id", nodeHandler.RemoveAlgorithm)
		nodes.GET("/recommend-node", nodeHandler.RecommendNode)

		// Node tag management
		nodes.PUT("/:id/tags", nodeHandler.UpdateTags)
	}

	// Edge node tag routes
	if deps.EdgeNodeTagHandler != nil {
		tags := authorized.Group("/edge-node-tags")
		tags.Use(r.RBAC())
		{
			tags.GET("", deps.EdgeNodeTagHandler.List)
			tags.GET("/all", deps.EdgeNodeTagHandler.ListAll)
			tags.POST("", deps.EdgeNodeTagHandler.Create)
			tags.GET("/:id", deps.EdgeNodeTagHandler.GetByID)
			tags.PUT("/:id", deps.EdgeNodeTagHandler.Update)
			tags.DELETE("/:id", deps.EdgeNodeTagHandler.Delete)
		}
	}

	// Edge node metrics routes
	if deps.EdgeNodeMetricsHandler != nil {
		metrics := authorized.Group("/edge-nodes")
		metrics.Use(r.RBAC())
		{
			metrics.GET("/overview", deps.EdgeNodeMetricsHandler.GetNodeOverview)
			metrics.GET("/:node_id/metrics", deps.EdgeNodeMetricsHandler.QueryMetrics)
		}
	}

	// Terminal session routes
	if deps.TerminalHandler != nil {
		// WebSocket terminal — JWT auth handled internally, like /ws
		v1.GET("/ws/terminal", deps.TerminalHandler.HandleWebSocket)

		// Session REST API
		sessions := authorized.Group("/edge-nodes/:id/sessions")
		sessions.Use(r.RBAC())
		{
			sessions.GET("", deps.TerminalHandler.ListSessions)
			sessions.GET("/:session_id", deps.TerminalHandler.GetSession)
			sessions.DELETE("/:session_id", deps.TerminalHandler.CloseSession)
			sessions.GET("/:session_id/recording", deps.TerminalHandler.GetRecording)
		}
	}

	// Edge scheduled task routes
	if deps.EdgeScheduledTaskHandler != nil {
		scheduled := authorized.Group("/edge-scheduled-tasks")
		scheduled.Use(r.RBAC())
		{
			scheduled.GET("", deps.EdgeScheduledTaskHandler.List)
			scheduled.POST("", deps.EdgeScheduledTaskHandler.Create)
			scheduled.GET("/records", deps.EdgeScheduledTaskHandler.ListRecords)
			scheduled.GET("/:id", deps.EdgeScheduledTaskHandler.GetByID)
			scheduled.PUT("/:id", deps.EdgeScheduledTaskHandler.Update)
			scheduled.DELETE("/:id", deps.EdgeScheduledTaskHandler.Delete)
			scheduled.PUT("/:id/toggle", deps.EdgeScheduledTaskHandler.ToggleEnabled)
			scheduled.POST("/:id/records/:record_id/retry", deps.EdgeScheduledTaskHandler.RetryRecord)
		}
	}
}
