package handler

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/pkg/response"
	"github.com/niko-admin/niko-admin/internal/repository"
	"github.com/niko-admin/niko-admin/internal/service"
)

// EdgeNodeMetricsHandler 边缘节点指标 HTTP 处理器
type EdgeNodeMetricsHandler struct {
	svc *service.EdgeNodeMetricsService
}

// NewEdgeNodeMetricsHandler 创建 EdgeNodeMetricsHandler
func NewEdgeNodeMetricsHandler(svc *service.EdgeNodeMetricsService) *EdgeNodeMetricsHandler {
	return &EdgeNodeMetricsHandler{svc: svc}
}

// QueryMetrics 查询节点时序指标
//
// @Summary      查询节点时序指标
// @Description  查询节点时序指标，支持聚合和时间窗口降采样
// @Tags         节点指标
// @Produce      json
// @Param        node_id      path   string  true   "节点ID"
// @Param        metric       query  string  true   "指标名(cpu_usage/memory_usage/...)"
// @Param        from         query  string  true   "起始时间(RFC3339)"
// @Param        to           query  string  true   "结束时间(RFC3339)"
// @Param        aggregation  query  string  false  "聚合方式(avg/max/min)"
// @Param        interval     query  string  false  "聚合窗口(5m/1h/1d)"
// @Param        page         query  int     false  "页码"  default(1)
// @Param        page_size    query  int     false  "每页数量"  default(500)
// @Success      200  {object}  dto.Response{data=dto.PageData{list=[]map[string]interface{}}}
// @Router       /edge-nodes/{node_id}/metrics [get]
// @Security     BearerAuth
func (h *EdgeNodeMetricsHandler) QueryMetrics(c *gin.Context) {
	nodeID := c.Param("node_id")
	var req dto.MetricsQueryRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Err(c, badRequestError(c, err))
		return
	}

	from, err := time.Parse(time.RFC3339, req.From)
	if err != nil {
		response.Err(c, badRequestError(c, err))
		return
	}
	to, err := time.Parse(time.RFC3339, req.To)
	if err != nil {
		response.Err(c, badRequestError(c, err))
		return
	}

	opts := repository.MetricsQueryOpts{
		NodeID:      nodeID,
		Metric:      req.Metric,
		From:        from,
		To:          to,
		Aggregation: req.Aggregation,
		Interval:    req.Interval,
		Page:        req.GetPage(),
		PageSize:    req.GetPageSize(),
	}

	items, total, err := h.svc.QueryMetrics(c.Request.Context(), opts)
	if err != nil {
		response.Err(c, err)
		return
	}

	response.Page(c, items, total, req.GetPage(), req.GetPageSize())
}

// GetNodeOverview 获取节点总览统计
//
// @Summary      节点总览统计
// @Description  返回在线/离线/错误/禁用节点数量
// @Tags         节点指标
// @Produce      json
// @Success      200  {object}  dto.Response{data=dto.NodeOverviewResponse}
// @Router       /edge-nodes/overview [get]
// @Security     BearerAuth
func (h *EdgeNodeMetricsHandler) GetNodeOverview(c *gin.Context) {
	resp, err := h.svc.GetNodeOverview(c.Request.Context())
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, resp)
}
