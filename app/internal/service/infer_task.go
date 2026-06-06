package service

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/niko-admin/niko-admin/internal/model"
	"github.com/niko-admin/niko-admin/internal/pkg/errors"
)

// inferTaskRepo 推理任务持久化接口
type inferTaskRepo interface {
	FindByID(ctx context.Context, id string) (*model.InferTask, error)
	FindByDeviceID(ctx context.Context, deviceID string) (*model.InferTask, error)
	Create(ctx context.Context, item *model.InferTask) error
	UpdateStatus(ctx context.Context, id, status, errorCode, errorMessage string) error
}

// InferTaskService 推理任务管理
type InferTaskService struct {
	repo          inferTaskRepo
	streamManager *StreamManager
}

func NewInferTaskService(repo inferTaskRepo, streamManager *StreamManager) *InferTaskService {
	return &InferTaskService{
		repo:          repo,
		streamManager: streamManager,
	}
}

// Create 创建推理任务并启动流
func (s *InferTaskService) Create(ctx context.Context, taskName, deviceID, streamURL string) (*model.InferTask, error) {
	// 1. 检查设备是否已有活跃推理任务
	existing, _ := s.repo.FindByDeviceID(ctx, deviceID)
	if existing != nil && existing.Status == model.InferTaskStatusRunning {
		return nil, errors.New(errors.ErrBadRequest, fmt.Sprintf("device %s already has a running infer task", deviceID))
	}

	// 2. 创建任务记录
	item := &model.InferTask{
		TaskName:  taskName,
		DeviceID:  deviceID,
		StreamURL: streamURL,
		Status:    model.InferTaskStatusPending,
	}
	if err := s.repo.Create(ctx, item); err != nil {
		return nil, err
	}

	// 3. 通过 StreamManager 申请推理流
	if err := s.streamManager.Acquire(ctx, deviceID, "infer", map[string]string{
		"task_id": item.ID,
	}); err != nil {
		if updateErr := s.repo.UpdateStatus(ctx, item.ID, model.InferTaskStatusFailed, "ACQUIRE_FAILED", err.Error()); updateErr != nil {
			zap.L().Error("failed to update infer task status to failed",
				zap.String("task_id", item.ID), zap.Error(updateErr))
		}
		return nil, err
	}

	// 4. 更新任务状态为运行中
	if err := s.repo.UpdateStatus(ctx, item.ID, model.InferTaskStatusRunning, "", ""); err != nil {
		zap.L().Error("failed to update infer task status to running",
			zap.String("task_id", item.ID), zap.Error(err))
	}

	return item, nil
}

// Stop 停止推理任务并释放流
func (s *InferTaskService) Stop(ctx context.Context, id string) error {
	task, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return errors.New(errors.ErrNotFound, "infer task not found")
	}

	// 通过 StreamManager 释放推理引用
	if err := s.streamManager.Release(ctx, task.DeviceID, "infer"); err != nil {
		return err
	}

	if err := s.repo.UpdateStatus(ctx, task.ID, model.InferTaskStatusStopped, "", ""); err != nil {
		zap.L().Error("failed to update infer task status to stopped",
			zap.String("task_id", task.ID), zap.Error(err))
	}
	return nil
}
