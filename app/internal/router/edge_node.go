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

		// Phase 1: Monitoring metrics routes
		nodes.GET("/overview", metricsHandler.Overview)
		nodes.GET("/:id/metrics", metricsHandler.QueryMetrics)

		// Phase 2: Alert rules routes
		nodes.POST("/:id/alert-rules", alertRuleHandler.List) // List rules for a node
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
