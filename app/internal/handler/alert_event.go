package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/niko-admin/niko-admin/internal/dto"
	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
	"github.com/niko-admin/niko-admin/internal/pkg/response"
	"github.com/niko-admin/niko-admin/internal/service"
)

// AlertEventHandler handles HTTP requests for AlertEvent operations.
type AlertEventHandler struct {
	svc *service.AlertEventService
}

// NewAlertEventHandler creates a new AlertEventHandler.
func NewAlertEventHandler(svc *service.AlertEventService) *AlertEventHandler {
	return &AlertEventHandler{svc: svc}
}

// List returns a paginated list of alert events.
//
// @Summary      告警事件列表
// @Description  分页查询告警事件列表，支持按节点、规则、状态和时间范围筛选
// @Tags         告警事件
// @Produce      json
// @Param        page      query   int     false  "页码"      default(1)
// @Param        page_size query   int     false  "每页数量"  default(20)
// @Param        node_id   query   string  false  "节点ID筛选"
// @Param        rule_id   query   string  false  "规则ID筛选"
// @Param        status    query   string  false  "状态筛选(firing/resolved/acknowledged)"
// @Param        from      query   string  false  "起始时间(RFC3339)"
// @Param        to        query   string  false  "结束时间(RFC3339)"
// @Success      200  {object}  dto.Response{data=dto.PageData{list=[]dto.AlertEventResponse}}
// @Router       /alert-events [get]
// @Security     BearerAuth
func (h *AlertEventHandler) List(c *gin.Context) {
	var req dto.AlertEventListRequest
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

// Acknowledge acknowledges an alert event.
//
// @Summary      确认告警事件
// @Description  手动确认一个告警事件
// @Tags         告警事件
// @Accept       json
// @Produce      json
// @Param        id    path   string                          true  "事件 ID"
// @Param        body  body   dto.AcknowledgeAlertEventRequest true  "确认信息"
// @Success      200   {object}  dto.Response
// @Router       /alert-events/{id}/acknowledge [post]
// @Security     BearerAuth
func (h *AlertEventHandler) Acknowledge(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Err(c, apperrors.NewLocalized(apperrors.ErrBadRequest, "缺少告警事件 ID"))
		return
	}

	var req dto.AcknowledgeAlertEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Err(c, badRequestError(c, err))
		return
	}

	if err := h.svc.Acknowledge(c.Request.Context(), id, req.AcknowledgedBy); err != nil {
		response.Err(c, err)
		return
	}

	response.OK(c, nil)
}
