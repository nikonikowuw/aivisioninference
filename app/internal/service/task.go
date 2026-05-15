package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/model"
	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
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
func (s *TaskService) List(ctx context.Context, page, pageSize int, taskType, status string) ([]model.Task, int64, error) {
	return s.taskRepo.ListFiltered(ctx, page, pageSize, taskType, status)
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
