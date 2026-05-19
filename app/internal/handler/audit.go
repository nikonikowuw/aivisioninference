package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/middleware"
	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
	"github.com/niko-admin/niko-admin/internal/pkg/response"
	"github.com/niko-admin/niko-admin/internal/service"
)

// AuditHandler handles audit log HTTP requests.
type AuditHandler struct {
	svc *service.AuditService
}

// NewAuditHandler creates a new AuditHandler with the given dependencies.
func NewAuditHandler(svc *service.AuditService) *AuditHandler {
	return &AuditHandler{svc: svc}
}

// List returns a paginated list of audit logs with optional filters.
//
// @Summary      审计日志列表
// @Description  分页查询审计日志，支持按关键词、操作、资源类型筛选
// @Tags         审计日志
// @Produce      json
// @Param        page          query   int     false  "页码"        default(1)
// @Param        page_size     query   int     false  "每页数量"    default(20)
// @Param        keyword       query   string  false  "关键词搜索（用户名/操作/资源）"
// @Param        action        query   string  false  "操作类型筛选"
// @Param        resource_type query   string  false  "资源类型筛选"
// @Param        start_time    query   string  false  "开始时间 (RFC3339)"  Format(date-time)
// @Param        end_time      query   string  false  "结束时间 (RFC3339)"  Format(date-time)
// @Success      200  {object}  dto.Response{data=dto.PageData{list=[]dto.AuditLogResponse}}
// @Router       /audit-logs [get]
// @Security     BearerAuth
func (h *AuditHandler) List(c *gin.Context) {
	var req dto.ListAuditLogRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		attachError(c, apperrors.New(apperrors.ErrBadRequest, err.Error()))
		return
	}

	lang, _ := c.Get(middleware.ContextKeyLang)
	langStr, _ := lang.(string)

	result, err := h.svc.List(c.Request.Context(), langStr, req)
	if err != nil {
		attachError(c, err)
		return
	}

	response.Page(c, result.List, result.Total, req.GetPage(), req.GetPageSize())
}
