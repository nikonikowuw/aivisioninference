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

// TaskService 处理异步任务相关的业务逻辑
type TaskService struct {
	taskRepo *repository.TaskRepository // 任务数据持久化接口
}

// NewTaskService 创建并返回一个新的 TaskService 实例
func NewTaskService(taskRepo *repository.TaskRepository) *TaskService {
	return &TaskService{taskRepo: taskRepo}
}

// Create 创建并排队一个新的异步后台任务
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

// List 根据分页和可选的过滤条件返回任务列表和总条数
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

// GetByID 根据任务 ID 查询任务详情
func (s *TaskService) GetByID(ctx context.Context, id string) (*model.Task, error) {
	task, err := s.taskRepo.FindByID(ctx, id)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrNotFound, "任务不存在")
	}
	return task, nil
}

// Cancel 取消处于排队中（pending）或执行中（running）的后台任务
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
