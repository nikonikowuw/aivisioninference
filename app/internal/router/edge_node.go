package router

import (
	"github.com/gin-gonic/gin"
)

func (r *Router) registerEdgeNodeRoutes(authorized *gin.RouterGroup, v1 *gin.RouterGroup, deps *RouteDeps) {
	nodeHandler := deps.EdgeNodeHandler
	nodeMiddleware := deps.EdgeNodeMiddleware
	metricsHandler := deps.EdgeNodeMetricsHandler
	alertRuleHandler := deps.AlertRuleHandler
	alertEventHandler := deps.AlertEventHandler
	scheduledTaskHandler := deps.EdgeNodeScheduledTaskHandler
	terminalHandler := deps.EdgeNodeTerminalHandler

	// Ensure mandatory handlers are non-nil — a nil value here is a startup bug,
	// not a runtime condition, so panic is the correct response.
	if nodeHandler == nil {
		panic("nodeHandler is nil — check Wire provider for EdgeNodeHandler")
	}
	if metricsHandler == nil {
		panic("metricsHandler is nil — check Wire provider for EdgeNodeMetricsHandler")
	}
	if alertRuleHandler == nil {
		panic("alertRuleHandler is nil — check Wire provider for AlertRuleHandler")
	}
	if scheduledTaskHandler == nil {
		panic("scheduledTaskHandler is nil — check Wire provider for EdgeNodeScheduledTaskHandler")
	}
	if terminalHandler == nil {
		panic("terminalHandler is nil — check Wire provider for EdgeNodeTerminalHandler")
	}

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

	// Phase 1: Monitoring metrics routes
	nodes.GET("/overview", metricsHandler.Overview)
	nodes.GET("/:id/metrics", metricsHandler.QueryMetrics)

	// Phase 2: Alert rules routes
	nodes.POST("/:id/alert-rules", alertRuleHandler.List)

	// Phase 3: Scheduled task routes
	nodes.GET("/:id/scheduled-tasks", scheduledTaskHandler.List)
	nodes.POST("/:id/scheduled-tasks/create", scheduledTaskHandler.Create)
	nodes.GET("/:id/scheduled-tasks/:task_id", scheduledTaskHandler.GetByID)
	nodes.PUT("/:id/scheduled-tasks/:task_id", scheduledTaskHandler.Update)
	nodes.DELETE("/:id/scheduled-tasks/:task_id", scheduledTaskHandler.Delete)

	// Phase 3: Task execution listing
	nodes.GET("/:id/task-executions", scheduledTaskHandler.ListExecutions)

	// Phase 3: Web Terminal (WebSocket)
	nodes.GET("/:id/terminal", terminalHandler.HandleWebSocket)

	// Phase 3: Task execution callback (no auth — called by engine via HTTP)
	v1.POST("/edge-nodes/:id/task-executions/callback", scheduledTaskHandler.HandleCallback)

	// Terminal session routes (HEAD original — kept for backward compatibility)
	if deps.TerminalHandler != nil {
		v1.GET("/ws/terminal", deps.TerminalHandler.HandleWebSocket)

		sessions := authorized.Group("/edge-nodes/:id/sessions")
		sessions.Use(r.RBAC())
		{
			sessions.GET("", deps.TerminalHandler.ListSessions)
			sessions.GET("/:session_id", deps.TerminalHandler.GetSession)
			sessions.DELETE("/:session_id", deps.TerminalHandler.CloseSession)
			sessions.GET("/:session_id/recording", deps.TerminalHandler.GetRecording)
		}
	}

	// Edge scheduled task routes (HEAD original — kept for backward compatibility)
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

	// Phase 2: Alert Rules (admin management)
	alertRules := authorized.Group("/alert-rules")
	alertRules.Use(r.RBAC())
	{
		alertRules.GET("", alertRuleHandler.List)
		alertRules.POST("", alertRuleHandler.Create)
		alertRules.GET("/:id", alertRuleHandler.GetByID)
		alertRules.PUT("/:id", alertRuleHandler.Update)
		alertRules.DELETE("/:id", alertRuleHandler.Delete)
	}

	// Phase 2: Alert Events
	alertEvents := authorized.Group("/alert-events")
	alertEvents.Use(r.RBAC())
	{
		alertEvents.GET("", alertEventHandler.List)
		alertEvents.POST("/:id/acknowledge", alertEventHandler.Acknowledge)
	}
}
