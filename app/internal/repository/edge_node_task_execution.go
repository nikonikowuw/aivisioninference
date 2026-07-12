package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/niko-admin/niko-admin/internal/model"
)

// EdgeNodeTaskExecutionRepository handles DB operations for EdgeNodeTaskExecution.
type EdgeNodeTaskExecutionRepository struct {
	db *gorm.DB
}

// NewEdgeNodeTaskExecutionRepository creates a new repository.
func NewEdgeNodeTaskExecutionRepository(db *gorm.DB) *EdgeNodeTaskExecutionRepository {
	return &EdgeNodeTaskExecutionRepository{db: db}
}

// Create inserts a new task execution record.
func (r *EdgeNodeTaskExecutionRepository) Create(ctx context.Context, exec *model.EdgeNodeTaskExecution) error {
	return r.db.WithContext(ctx).Create(exec).Error
}

// FindByID finds a task execution by ID.
func (r *EdgeNodeTaskExecutionRepository) FindByID(ctx context.Context, id string) (*model.EdgeNodeTaskExecution, error) {
	var exec model.EdgeNodeTaskExecution
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&exec).Error
	if err != nil {
		return nil, err
	}
	return &exec, nil
}

// ListByTask lists execution records for a task with pagination.
func (r *EdgeNodeTaskExecutionRepository) ListByTask(ctx context.Context, taskID string, page, pageSize int, status string) ([]model.EdgeNodeTaskExecution, int64, error) {
	query := r.db.WithContext(ctx).Model(&model.EdgeNodeTaskExecution{}).Where("task_id = ?", taskID)
	if status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var executions []model.EdgeNodeTaskExecution
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&executions).Error; err != nil {
		return nil, 0, err
	}

	return executions, total, nil
}

// GetLatestByTask returns the most recent execution record for a task.
func (r *EdgeNodeTaskExecutionRepository) GetLatestByTask(ctx context.Context, taskID string) (*model.EdgeNodeTaskExecution, error) {
	var exec model.EdgeNodeTaskExecution
	err := r.db.WithContext(ctx).
		Where("task_id = ?", taskID).
		Order("created_at DESC").
		First(&exec).Error
	if err != nil {
		return nil, err
	}
	return &exec, nil
}

// UpdateResult updates the execution result fields.
func (r *EdgeNodeTaskExecutionRepository) UpdateResult(ctx context.Context, id string, status string, stdout, stderr string, exitCode int, finishedAt time.Time) error {
	return r.db.WithContext(ctx).Model(&model.EdgeNodeTaskExecution{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":      status,
			"stdout":      stdout,
			"stderr":      stderr,
			"exit_code":   exitCode,
			"finished_at": finishedAt,
		}).Error
}

// UpdateStatus updates only the status of an execution record.
func (r *EdgeNodeTaskExecutionRepository) UpdateStatus(ctx context.Context, id, status string) error {
	return r.db.WithContext(ctx).Model(&model.EdgeNodeTaskExecution{}).
		Where("id = ?", id).
		Update("status", status).Error
}
