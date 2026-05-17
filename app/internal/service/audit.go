package service

import (
	"context"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/model"
	"github.com/niko-admin/niko-admin/internal/pkg/scopes"
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

// List returns a paginated list of audit logs with optional filters.
//
// 【核心功能】查询审计日志列表，支持关键字、操作类型、资源类型、时间范围等筛选条件。
// 时间范围参数通过公共 ParseTimeRange 方法解析，确保格式统一。
func (s *AuditService) List(ctx context.Context, req dto.ListAuditLogRequest) (*ListResult, error) {
	// 解析时间范围参数
	if req.StartTime != "" || req.EndTime != "" {
		from, to, err := scopes.ParseTimeRange(req.StartTime, req.EndTime)
		if err != nil {
			return nil, mapTimeRangeError(err)
		}
		req.FromTime = from
		req.ToTime = to
	}

	logs, total, err := s.auditRepo.List(ctx, req)
	if err != nil {
		return nil, err
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
			UserAgent:      l.UserAgent,
			ResponseStatus: l.ResponseStatus,
			DurationMs:     l.DurationMs,
			ResultSummary:  l.ResultSummary,
			ErrorSummary:   l.ErrorSummary,
			CreatedAt:      l.CreatedAt,
		})
	}

	return &ListResult{List: list, Total: total}, nil
}

// Create inserts a new audit log record.
func (s *AuditService) Create(ctx context.Context, log *model.AuditLog) error {
	return s.auditRepo.Create(ctx, log)
}
