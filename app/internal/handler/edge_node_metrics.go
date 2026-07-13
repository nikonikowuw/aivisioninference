package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/pkg/response"
	"github.com/niko-admin/niko-admin/internal/service"
)

// EdgeNodeMetricsHandler handles HTTP requests for edge node metrics operations.
type EdgeNodeMetricsHandler struct {
	svc         *service.EdgeNodeMetricsService
	edgeNodeSvc *service.EdgeNodeService
}

// NewEdgeNodeMetricsHandler creates a new EdgeNodeMetricsHandler.
func NewEdgeNodeMetricsHandler(svc *service.EdgeNodeMetricsService, edgeNodeSvc *service.EdgeNodeService) *EdgeNodeMetricsHandler {
	return &EdgeNodeMetricsHandler{
		svc:         svc,
		edgeNodeSvc: edgeNodeSvc,
	}
}

// QueryMetrics returns time-series metrics data for a specific edge node.
//
// @Summary      查询节点历史指标
// @Description  分页查询指定边缘节点的历史指标数据，支持时间范围、指标类型、聚合方式和分页
// @Tags         边缘节点指标
// @Produce      json
// @Param        id              path    string  true   "节点 ID"
// @Param        metric          query   string  true   "指标类型(cpu_usage/memory_usage/disk_usage/...)"
// @Param        from            query   string  false  "起始时间(RFC3339)，默认24小时前"
// @Param        to              query   string  false  "结束时间(RFC3339)，默认当前"
// @Param        aggregation     query   string  false  "聚合方式(avg/max/min)"
// @Param        interval        query   string  false  "聚合窗口(5m/1h/1d)"
// @Param        page            query   int     false  "页码"      default(1)
// @Param        page_size       query   int     false  "每页数量"  default(500)
// @Success      200  {object}  dto.Response{data=dto.MetricQueryResponse}
// @Router       /edge-nodes/{id}/metrics [get]
// @Security     BearerAuth
func (h *EdgeNodeMetricsHandler) QueryMetrics(c *gin.Context) {
	id := c.Param("id")

	var req dto.MetricQueryRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Err(c, badRequestError(c, err))
		return
	}

	res, err := h.svc.QueryMetrics(c.Request.Context(), id, req)
	if err != nil {
		response.Err(c, err)
		return
	}

	response.Page(c, res.List, res.Total, req.GetPage(), req.GetPageSize())
}

// Overview returns aggregated overview stats for all edge nodes.
//
// @Summary      节点总览统计
// @Description  返回所有边缘节点的在线/离线/错误节点数量和总任务数
// @Tags         边缘节点指标
// @Produce      json
// @Success      200  {object}  dto.Response{data=dto.OverviewStats}
// @Router       /edge-nodes/overview [get]
// @Security     BearerAuth
func (h *EdgeNodeMetricsHandler) Overview(c *gin.Context) {
	stats, err := h.edgeNodeSvc.GetOverviewStats(c.Request.Context())
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, stats)
}
