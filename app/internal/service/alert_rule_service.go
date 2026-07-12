package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/model"
	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
	"github.com/niko-admin/niko-admin/internal/repository"
)

// AlertRuleService handles business logic for AlertRule operations.
type AlertRuleService struct {
	ruleRepo *repository.AlertRuleRepository
}

// NewAlertRuleService creates a new AlertRuleService.
func NewAlertRuleService(ruleRepo *repository.AlertRuleRepository) *AlertRuleService {
	return &AlertRuleService{
		ruleRepo: ruleRepo,
	}
}

// Create creates a new alert rule.
func (s *AlertRuleService) Create(ctx context.Context, req dto.CreateAlertRuleRequest) (*model.AlertRule, error) {
	// Validate name uniqueness
	exists, err := s.ruleRepo.ExistsByName(ctx, req.Name, "")
	if err != nil {
		return nil, fmt.Errorf("检查规则名称唯一性失败: %w", err)
	}
	if exists {
		return nil, apperrors.New(apperrors.ErrBadRequest, "告警规则名称已存在")
	}

	// Serialize notify channels to JSON
	notifyChannels, err := json.Marshal(req.NotifyChannels)
	if err != nil {
		return nil, fmt.Errorf("序列化通知渠道失败: %w", err)
	}
	if req.NotifyChannels == nil {
		notifyChannels = []byte("[]")
	}

	now := time.Now()
	rule := &model.AlertRule{
		BaseModel:        model.BaseModel{ID: uuid.New().String()},
		Name:             req.Name,
		NodeID:           req.NodeID,
		MetricType:       req.MetricType,
		Operator:         req.Operator,
		Threshold:        req.Threshold,
		DurationSeconds:  req.DurationSeconds,
		SilenceMinutes:   req.SilenceMinutes,
		Enabled:          true,
		NotifyChannels:   notifyChannels,
		Description:      req.Description,
		UpdatedAt:        now,
	}

	if req.Enabled != nil {
		rule.Enabled = *req.Enabled
	}

	if err := s.ruleRepo.Create(ctx, rule); err != nil {
		return nil, fmt.Errorf("创建告警规则失败: %w", err)
	}

	return rule, nil
}

// Update updates an existing alert rule.
func (s *AlertRuleService) Update(ctx context.Context, id string, req dto.UpdateAlertRuleRequest) error {
	rule, err := s.ruleRepo.FindByID(ctx, id)
	if err != nil {
		return apperrors.New(apperrors.ErrNotFound, "告警规则不存在")
	}

	if req.Name != nil {
		exists, err := s.ruleRepo.ExistsByName(ctx, *req.Name, id)
		if err != nil {
			return fmt.Errorf("检查规则名称唯一性失败: %w", err)
		}
		if exists {
			return apperrors.New(apperrors.ErrBadRequest, "告警规则名称已存在")
		}
		rule.Name = *req.Name
	}

	if req.NodeID != nil {
		rule.NodeID = req.NodeID
	}
	if req.MetricType != nil {
		rule.MetricType = *req.MetricType
	}
	if req.Operator != nil {
		rule.Operator = *req.Operator
	}
	if req.Threshold != nil {
		rule.Threshold = *req.Threshold
	}
	if req.DurationSeconds != nil {
		rule.DurationSeconds = *req.DurationSeconds
	}
	if req.SilenceMinutes != nil {
		rule.SilenceMinutes = *req.SilenceMinutes
	}
	if req.Enabled != nil {
		rule.Enabled = *req.Enabled
	}
	if req.NotifyChannels != nil {
		channels, err := json.Marshal(req.NotifyChannels)
		if err != nil {
			return fmt.Errorf("序列化通知渠道失败: %w", err)
		}
		rule.NotifyChannels = channels
	}
	if req.Description != nil {
		rule.Description = *req.Description
	}

	rule.UpdatedAt = time.Now()

	return s.ruleRepo.Update(ctx, rule)
}

// Delete deletes an alert rule by ID.
func (s *AlertRuleService) Delete(ctx context.Context, id string) error {
	_, err := s.ruleRepo.FindByID(ctx, id)
	if err != nil {
		return apperrors.New(apperrors.ErrNotFound, "告警规则不存在")
	}

	return s.ruleRepo.Delete(ctx, id)
}

// GetByID returns a single alert rule by ID.
func (s *AlertRuleService) GetByID(ctx context.Context, id string) (*model.AlertRule, error) {
	rule, err := s.ruleRepo.FindByID(ctx, id)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrNotFound, "告警规则不存在")
	}
	return rule, nil
}

// List returns a paginated list of alert rules.
func (s *AlertRuleService) List(ctx context.Context, req dto.AlertRuleListRequest) ([]model.AlertRule, int64, error) {
	return s.ruleRepo.List(ctx, req)
}
