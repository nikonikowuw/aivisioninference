package service

import (
	"context"
	"encoding/json"
	"time"

	"gorm.io/datatypes"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/model"
	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
	"github.com/niko-admin/niko-admin/internal/repository"
)

// AITimeScheduleService 处理 AITimeSchedule 业务逻辑
type AITimeScheduleService struct {
	repo *repository.AITimeScheduleRepository
}

// NewAITimeScheduleService 创建新的 AITimeScheduleService
func NewAITimeScheduleService(repo *repository.AITimeScheduleRepository) *AITimeScheduleService {
	return &AITimeScheduleService{repo: repo}
}

// List 分页查询时间配置
func (s *AITimeScheduleService) List(ctx context.Context, req dto.AITimeScheduleListRequest) ([]model.AITimeSchedule, int64, error) {
	return s.repo.List(ctx, req)
}

// ListAll 返回全部时间配置（供前端下拉选择）
func (s *AITimeScheduleService) ListAll(ctx context.Context) ([]model.AITimeSchedule, error) {
	return s.repo.ListAll(ctx)
}

// Create 创建时间配置
func (s *AITimeScheduleService) Create(ctx context.Context, req dto.CreateAITimeScheduleRequest) (*model.AITimeSchedule, error) {
	startDate, err := time.Parse(time.DateOnly, req.StartDate)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrStartTimeFormat, "")
	}
	endDate, err := time.Parse(time.DateOnly, req.EndDate)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrEndTimeFormat, "")
	}
	if endDate.Before(startDate) {
		return nil, apperrors.New(apperrors.ErrTimeRangeOrder, "")
	}

	twBytes, err := json.Marshal(req.TimeWindows)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternal, "")
	}

	item := &model.AITimeSchedule{
		Name:        req.Name,
		Description: req.Description,
		StartDate:   datatypes.Date(startDate),
		EndDate:     datatypes.Date(endDate),
		TimeWindows: datatypes.JSON(twBytes),
	}
	if err := s.repo.Create(ctx, item); err != nil {
		return nil, err
	}
	return item, nil
}

// GetByID 查询单个时间配置
func (s *AITimeScheduleService) GetByID(ctx context.Context, id string) (*model.AITimeSchedule, error) {
	return s.repo.FindByID(ctx, id)
}

// Update 更新时间配置
func (s *AITimeScheduleService) Update(ctx context.Context, id string, req dto.UpdateAITimeScheduleRequest) error {
	item, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	startDate, err := time.Parse(time.DateOnly, req.StartDate)
	if err != nil {
		return apperrors.New(apperrors.ErrStartTimeFormat, "")
	}
	endDate, err := time.Parse(time.DateOnly, req.EndDate)
	if err != nil {
		return apperrors.New(apperrors.ErrEndTimeFormat, "")
	}
	if endDate.Before(startDate) {
		return apperrors.New(apperrors.ErrTimeRangeOrder, "")
	}

	twBytes, err := json.Marshal(req.TimeWindows)
	if err != nil {
		return apperrors.New(apperrors.ErrInternal, "")
	}

	item.Name = req.Name
	item.Description = req.Description
	item.StartDate = datatypes.Date(startDate)
	item.EndDate = datatypes.Date(endDate)
	item.TimeWindows = datatypes.JSON(twBytes)

	return s.repo.Update(ctx, item)
}

// Delete 删除时间配置
func (s *AITimeScheduleService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}
