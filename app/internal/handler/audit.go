package handler

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/model"
	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
	"github.com/niko-admin/niko-admin/internal/pkg/response"
)

// AuditHandler handles audit log HTTP requests.
type AuditHandler struct {
	db *gorm.DB
}

// NewAuditHandler creates a new AuditHandler with the given database.
func NewAuditHandler(db *gorm.DB) *AuditHandler {
	return &AuditHandler{db: db}
}

// List returns a paginated list of audit logs with optional filters.
//
// @Summary      审计日志列表
// @Description  分页查询审计日志，支持按用户、操作、资源类型、时间范围筛选
// @Tags         审计日志
// @Produce      json
// @Param        page          query   int     false  "页码"        default(1)
// @Param        page_size     query   int     false  "每页数量"    default(20)
// @Param        user_id       query   string  false  "用户 ID 筛选"
// @Param        action        query   string  false  "操作类型筛选"
// @Param        resource_type query   string  false  "资源类型筛选"
// @Param        start_time    query   string  false  "开始时间 (RFC3339)"
// @Param        end_time      query   string  false  "结束时间 (RFC3339)"
// @Success      200  {object}  dto.Response{data=dto.PageData{list=[]dto.AuditLogResponse}}
// @Router       /audit-logs [get]
// @Security     BearerAuth
func (h *AuditHandler) List(c *gin.Context) {
	var req dto.ListAuditLogRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Err(c, apperrors.New(apperrors.ErrBadRequest, err.Error()))
		return
	}

	page := req.GetPage()
	pageSize := req.GetPageSize()

	query := h.db.Model(&model.AuditLog{})

	// Apply filters
	if req.UserID != nil && *req.UserID != "" {
		query = query.Where("user_id = ?", *req.UserID)
	}
	if req.Action != "" {
		query = query.Where("action = ?", req.Action)
	}
	if req.ResourceType != "" {
		query = query.Where("resource_type = ?", req.ResourceType)
	}

	// Time range filters
	if req.StartTime != "" {
		startTime, err := time.Parse(time.RFC3339, req.StartTime)
		if err != nil {
			response.Err(c, apperrors.New(apperrors.ErrBadRequest, "start_time 格式错误，请使用 RFC3339"))
			return
		}
		query = query.Where("created_at >= ?", startTime)
	}
	if req.EndTime != "" {
		endTime, err := time.Parse(time.RFC3339, req.EndTime)
		if err != nil {
			response.Err(c, apperrors.New(apperrors.ErrBadRequest, "end_time 格式错误，请使用 RFC3339"))
			return
		}
		query = query.Where("created_at <= ?", endTime)
	}

	// Count total
	var total int64
	if err := query.Count(&total).Error; err != nil {
		zap.L().Error("count audit logs failed", zap.Error(err))
		response.Err(c, apperrors.New(apperrors.ErrInternal, ""))
		return
	}

	// Query logs
	var logs []model.AuditLog
	if err := query.Offset((page - 1) * pageSize).
		Limit(pageSize).
		Order("created_at DESC").
		Find(&logs).Error; err != nil {
		zap.L().Error("list audit logs failed", zap.Error(err))
		response.Err(c, apperrors.New(apperrors.ErrInternal, ""))
		return
	}

	// Convert to response DTOs
	list := make([]dto.AuditLogResponse, 0, len(logs))
	for _, l := range logs {
		list = append(list, dto.AuditLogResponse{
			ID:             l.ID,
			UserID:         l.UserID,
			Username:       l.Username,
			Action:         l.Action,
			ResourceType:   l.ResourceType,
			ResourceID:     l.ResourceID,
			RequestPath:    l.RequestPath,
			RequestMethod:  l.RequestMethod,
			RequestIP:      l.RequestIP,
			ResponseStatus: l.ResponseStatus,
			DurationMs:     l.DurationMs,
			CreatedAt:      l.CreatedAt,
		})
	}

	response.Page(c, list, total, page, pageSize)
}
