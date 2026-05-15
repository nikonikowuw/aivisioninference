package service

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/model"
	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
	"github.com/niko-admin/niko-admin/internal/repository"
)

// AuditService handles business logic for audit log operations.
type AuditService struct {
	auditRepo *repository.AuditRepository
}

// NewAuditService creates a new AuditService.
func NewAuditService(auditRepo *repository.AuditRepository) *AuditService {
	return &AuditService{auditRepo: auditRepo}
}

// ListResult holds the paginated audit log response.
type ListResult struct {
	List  []dto.AuditLogResponse
	Total int64
}

// List returns a paginated list of audit logs with typed filters.
func (s *AuditService) List(ctx context.Context, req dto.ListAuditLogRequest) (*ListResult, error) {
	page := req.GetPage()
	pageSize := req.GetPageSize()

	filters := repository.AuditLogFilters{
		Action:       req.Action,
		ResourceType: req.ResourceType,
	}

	if req.UserID != nil && *req.UserID != "" {
		filters.UserID = *req.UserID
	}

	if req.StartTime != "" {
		startTime, err := time.Parse(time.RFC3339, req.StartTime)
		if err != nil {
			return nil, apperrors.New(apperrors.ErrBadRequest, "start_time 格式错误，请使用 RFC3339")
		}
		filters.StartTime = &startTime
	}
	if req.EndTime != "" {
		endTime, err := time.Parse(time.RFC3339, req.EndTime)
		if err != nil {
			return nil, apperrors.New(apperrors.ErrBadRequest, "end_time 格式错误，请使用 RFC3339")
		}
		filters.EndTime = &endTime
	}

	logs, total, err := s.auditRepo.ListWithFilters(ctx, page, pageSize, filters)
	if err != nil {
		zap.L().Error("list audit logs failed", zap.Error(err))
		return nil, apperrors.New(apperrors.ErrInternal, "")
	}

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

	return &ListResult{List: list, Total: total}, nil
}

// Create inserts a new audit log record.
func (s *AuditService) Create(ctx context.Context, log *model.AuditLog) error {
	return s.auditRepo.Create(ctx, log)
}
