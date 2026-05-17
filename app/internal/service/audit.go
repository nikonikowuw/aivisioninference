package service

import (
	"context"
	"fmt"
	"time"

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

// List returns a paginated list of audit logs with optional filters.
func (s *AuditService) List(ctx context.Context, req dto.ListAuditLogRequest) (*ListResult, error) {
	if req.StartTime != "" {
		t, err := time.Parse(dto.DateTimeFormat, req.StartTime)
		if err != nil {
			return nil, apperrors.New(apperrors.ErrBadRequest, fmt.Sprintf("start_time format invalid, expected %s", dto.DateTimeFormat))
		}
		req.FromTime = &t
	}
	if req.EndTime != "" {
		t, err := time.Parse(dto.DateTimeFormat, req.EndTime)
		if err != nil {
			return nil, apperrors.New(apperrors.ErrBadRequest, fmt.Sprintf("end_time format invalid, expected %s", dto.DateTimeFormat))
		}
		req.ToTime = &t
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
