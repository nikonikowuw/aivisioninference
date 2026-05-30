// Package handler 提供 HTTP 请求处理层（Controller），负责参数绑定、校验和响应返回。
package handler

import (
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/pkg/response"
	"github.com/niko-admin/niko-admin/internal/service"
)

// AuditHandler 处理审计日志相关的 HTTP 请求。
type AuditHandler struct {
	svc *service.AuditService
}

const csvContentType = "text/csv; charset=utf-8"

var unsafeFilenameChars = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

func writeCSV(c *gin.Context, filename string, data []byte) {
	safeName := safeCSVFilename(filename)
	encodedName := url.PathEscape(safeName)
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Cache-Control", "no-store")
	c.Header("Content-Disposition", fmt.Sprintf(`%s; filename*=UTF-8''%s`, mime.FormatMediaType("attachment", map[string]string{"filename": safeName}), encodedName))
	c.Data(http.StatusOK, csvContentType, data)
}

func safeCSVFilename(filename string) string {
	filename = strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(filename, "\r", ""), "\n", ""))
	if filename == "" {
		return "export.csv"
	}
	filename = unsafeFilenameChars.ReplaceAllString(filename, "_")
	if !strings.HasSuffix(strings.ToLower(filename), ".csv") {
		filename += ".csv"
	}
	return filename
}

// NewAuditHandler 创建一个新的 AuditHandler 实例，注入相关的服务依赖。
func NewAuditHandler(svc *service.AuditService) *AuditHandler {
	return &AuditHandler{svc: svc}
}

// List 返回分页的审计日志列表，支持可选的过滤条件。
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
		attachError(c, badRequestError(c, err))
		return
	}

	result, err := h.svc.List(c.Request.Context(), currentLang(c), req)
	if err != nil {
		attachError(c, err)
		return
	}

	// 统一返回分页数据格式
	response.Page(c, result.List, result.Total, req.GetPage(), req.GetPageSize())
}

// ExportCSV 按当前筛选条件导出审计日志 CSV。
func (h *AuditHandler) ExportCSV(c *gin.Context) {
	var req dto.ListAuditLogRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		attachError(c, badRequestError(c, err))
		return
	}
	data, err := h.svc.ExportCSV(c.Request.Context(), currentLang(c), req)
	if err != nil {
		attachError(c, err)
		return
	}
	writeCSV(c, "audit-logs.csv", data)
}
