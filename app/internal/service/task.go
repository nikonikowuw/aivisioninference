package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/model"
	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
	"github.com/niko-admin/niko-admin/internal/pkg/scopes"
	"github.com/niko-admin/niko-admin/internal/repository"
)

// TaskService handles business logic for Task operations.
type TaskService struct {
	taskRepo *repository.TaskRepository
}

// NewTaskService creates a new TaskService.
func NewTaskService(taskRepo *repository.TaskRepository) *TaskService {
	return &TaskService{taskRepo: taskRepo}
}

// Create creates a new background task.
func (s *TaskService) Create(ctx context.Context, req dto.CreateTaskRequest) (*model.Task, error) {
	item := model.Task{
		TaskID:     uuid.New().String(),
		Type:       req.Type,
		Payload:    req.Payload,
		Status:     "pending",
		MaxRetries: 3,
	}

	if err := s.taskRepo.Create(ctx, &item); err != nil {
		zap.L().Error("create task failed", zap.Error(err))
		return nil, apperrors.New(apperrors.ErrInternal, "")
	}

	return &item, nil
}

// List returns a paginated list of tasks with optional filters.
//
// 【核心功能】查询任务列表，支持关键字、类型、状态、时间范围等筛选条件。
// 时间范围参数通过公共 ParseTimeRange 方法解析，确保格式统一。
func (s *TaskService) List(ctx context.Context, req dto.TaskListRequest) ([]model.Task, int64, error) {
	// 解析时间范围参数
	if req.StartTime != "" || req.EndTime != "" {
		from, to, err := scopes.ParseTimeRange(req.StartTime, req.EndTime)
		if err != nil {
			return nil, 0, mapTimeRangeError(err)
		}
		req.FromTime = from
		req.ToTime = to
	}
	return s.taskRepo.List(ctx, req)
}

// GetByID returns a task by its ID.
func (s *TaskService) GetByID(ctx context.Context, id string) (*model.Task, error) {
	task, err := s.taskRepo.FindByID(ctx, id)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrNotFound, "任务不存在")
	}
	return task, nil
}

// Cancel cancels a pending or running task.
func (s *TaskService) Cancel(ctx context.Context, id string) error {
	task, err := s.taskRepo.FindByID(ctx, id)
	if err != nil {
		return apperrors.New(apperrors.ErrNotFound, "任务不存在")
	}

	if task.Status != "pending" && task.Status != "running" {
		return apperrors.New(apperrors.ErrBadRequest, "任务状态不允许取消")
	}

	now := time.Now()
	if err := s.taskRepo.UpdateStatus(ctx, id, "cancelled", &now); err != nil {
		zap.L().Error("cancel task failed", zap.String("task_id", id), zap.Error(err))
		return apperrors.New(apperrors.ErrInternal, "")
	}

	return nil
}
