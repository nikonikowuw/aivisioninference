package service

import (
	"context"

	"github.com/niko-admin/niko-admin/internal/dto"
	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
	"github.com/niko-admin/niko-admin/internal/repository"
)

// AlertEventService handles business logic for AlertEvent operations.
type AlertEventService struct {
	eventRepo *repository.AlertEventRepository
}

// NewAlertEventService creates a new AlertEventService.
func NewAlertEventService(eventRepo *repository.AlertEventRepository) *AlertEventService {
	return &AlertEventService{
		eventRepo: eventRepo,
	}
}

// List returns a paginated list of alert events with joined fields.
func (s *AlertEventService) List(ctx context.Context, req dto.AlertEventListRequest) ([]dto.AlertEventResponse, int64, error) {
	return s.eventRepo.List(ctx, req)
}

// Acknowledge marks an alert event as acknowledged.
func (s *AlertEventService) Acknowledge(ctx context.Context, id string, acknowledgedBy string) error {
	// Check if event exists
	event, err := s.eventRepo.FindByID(ctx, id)
	if err != nil {
		return apperrors.New(apperrors.ErrNotFound, "告警事件不存在")
	}

	if event.Status == "acknowledged" {
		return apperrors.New(apperrors.ErrBadRequest, "告警事件已被确认")
	}

	return s.eventRepo.AcknowledgeEvent(ctx, id, acknowledgedBy)
}
