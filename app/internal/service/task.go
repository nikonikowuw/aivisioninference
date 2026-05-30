// Package service 提供业务逻辑层实现，包含认证鉴权、资源管理和系统配置等核心业务流程。
package service

import (
	"context"
	"strconv"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/model"
	"github.com/niko-admin/niko-admin/internal/pkg/csvx"
	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
	"github.com/niko-admin/niko-admin/internal/pkg/timex"
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

// List 查询任务列表，支持关键字、类型、状态、时间范围等筛选条件。
func (s *TaskService) List(ctx context.Context, req dto.TaskListRequest) ([]model.Task, int64, error) {
	if err := normalizeTaskTimeRange(&req); err != nil {
		return nil, 0, err
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

// BatchCancel 批量取消任务，逐条复用单条取消状态校验。
func (s *TaskService) BatchCancel(ctx context.Context, ids []string, lang string) dto.BatchResult {
	return runBatch(ids, lang, func(id string) error {
		return s.Cancel(ctx, id)
	})
}

// ExportCSV 导出当前筛选条件下的任务列表 CSV。
func (s *TaskService) ExportCSV(ctx context.Context, req dto.TaskListRequest) ([]byte, error) {
	if err := normalizeTaskTimeRange(&req); err != nil {
		return nil, err
	}
	items, err := s.taskRepo.ListForExport(ctx, req, maxCSVExportRows)
	if err != nil {
		zap.L().Error("export tasks failed", zap.Error(err))
		return nil, apperrors.New(apperrors.ErrInternal, "")
	}
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			item.ID,
			item.TaskID,
			item.Type,
			item.Status,
			strconv.Itoa(item.RetryCount),
			item.ErrorMessage,
			item.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	data, err := csvx.Build([]string{"ID", "TaskID", "Type", "Status", "RetryCount", "ErrorMessage", "CreatedAt"}, rows)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternal, "")
	}
	return data, nil
}

func normalizeTaskTimeRange(req *dto.TaskListRequest) error {
	return timex.NormalizeRange(req.StartTime, req.EndTime, &req.FromTime, &req.ToTime)
}
