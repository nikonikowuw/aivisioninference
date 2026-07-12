package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/niko-admin/niko-admin/internal/pkg/response"
	"github.com/niko-admin/niko-admin/internal/service"

	"github.com/niko-admin/niko-admin/internal/dto"
)

// AlertRuleHandler handles HTTP requests for AlertRule operations.
type AlertRuleHandler struct {
	svc *service.AlertRuleService
}

// NewAlertRuleHandler creates a new AlertRuleHandler.
func NewAlertRuleHandler(svc *service.AlertRuleService) *AlertRuleHandler {
	return &AlertRuleHandler{svc: svc}
}

// List returns a paginated list of alert rules.
//
// @Summary      告警规则列表
// @Description  分页查询告警规则列表
// @Tags         告警规则
// @Produce      json
// @Param        page        query   int     false  "页码"      default(1)
// @Param        page_size   query   int     false  "每页数量"  default(20)
// @Param        keyword     query   string  false  "关键词搜索"
// @Param        node_id     query   string  false  "节点ID筛选"
// @Param        metric_type query   string  false  "指标类型(cpu_usage/memory_usage/...)"
// @Param        enabled     query   string  false  "启用状态(true/false)"
// @Success      200  {object}  dto.Response{data=dto.PageData{list=[]model.AlertRule}}
// @Router       /alert-rules [get]
// @Security     BearerAuth
func (h *AlertRuleHandler) List(c *gin.Context) {
	var req dto.AlertRuleListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Err(c, badRequestError(c, err))
		return
	}

	items, total, err := h.svc.List(c.Request.Context(), req)
	if err != nil {
		response.Err(c, err)
		return
	}

	response.Page(c, items, total, req.GetPage(), req.GetPageSize())
}

// Create creates a new alert rule.
//
// @Summary      创建告警规则
// @Description  创建新的告警规则
// @Tags         告警规则
// @Accept       json
// @Produce      json
// @Param        body  body  dto.CreateAlertRuleRequest  true  "告警规则信息"
// @Success      200   {object}  dto.Response{data=model.AlertRule}
// @Router       /alert-rules [post]
// @Security     BearerAuth
func (h *AlertRuleHandler) Create(c *gin.Context) {
	var req dto.CreateAlertRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Err(c, badRequestError(c, err))
		return
	}

	rule, err := h.svc.Create(c.Request.Context(), req)
	if err != nil {
		response.Err(c, err)
		return
	}

	response.OK(c, rule)
}

// GetByID returns a single alert rule by ID.
//
// @Summary      获取告警规则详情
// @Description  根据 ID 查询告警规则详情
// @Tags         告警规则
// @Produce      json
// @Param        id   path   string  true  "规则 ID"
// @Success      200  {object}  dto.Response{data=model.AlertRule}
// @Router       /alert-rules/{id} [get]
// @Security     BearerAuth
func (h *AlertRuleHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	rule, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Err(c, err)
		return
	}

	response.OK(c, rule)
}

// Update updates an alert rule.
//
// @Summary      更新告警规则
// @Description  更新告警规则配置信息
// @Tags         告警规则
// @Accept       json
// @Produce      json
// @Param        id    path   string                    true  "规则 ID"
// @Param        body  body   dto.UpdateAlertRuleRequest  true  "更新信息"
// @Success      200   {object}  dto.Response
// @Router       /alert-rules/{id} [put]
// @Security     BearerAuth
func (h *AlertRuleHandler) Update(c *gin.Context) {
	id := c.Param("id")

	var req dto.UpdateAlertRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Err(c, badRequestError(c, err))
		return
	}

	if err := h.svc.Update(c.Request.Context(), id, req); err != nil {
		response.Err(c, err)
		return
	}

	response.OK(c, nil)
}

// Delete deletes an alert rule.
//
// @Summary      删除告警规则
// @Description  删除告警规则
// @Tags         告警规则
// @Produce      json
// @Param        id  path  string  true  "规则 ID"
// @Success      200  {object}  dto.Response
// @Router       /alert-rules/{id} [delete]
// @Security     BearerAuth
func (h *AlertRuleHandler) Delete(c *gin.Context) {
	id := c.Param("id")

	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		response.Err(c, err)
		return
	}

	response.OK(c, nil)
}
