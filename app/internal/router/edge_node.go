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
	}
}
