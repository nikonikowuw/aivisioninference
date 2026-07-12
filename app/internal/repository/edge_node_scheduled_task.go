package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/niko-admin/niko-admin/internal/model"
)

// EdgeNodeScheduledTaskRepository handles DB operations for EdgeNodeScheduledTask.
type EdgeNodeScheduledTaskRepository struct {
	db *gorm.DB
}

// NewEdgeNodeScheduledTaskRepository creates a new repository.
func NewEdgeNodeScheduledTaskRepository(db *gorm.DB) *EdgeNodeScheduledTaskRepository {
	return &EdgeNodeScheduledTaskRepository{db: db}
}

// Create inserts a new scheduled task.
func (r *EdgeNodeScheduledTaskRepository) Create(ctx context.Context, task *model.EdgeNodeScheduledTask) error {
	return r.db.WithContext(ctx).Create(task).Error
}

// FindByID finds a scheduled task by ID.
func (r *EdgeNodeScheduledTaskRepository) FindByID(ctx context.Context, id string) (*model.EdgeNodeScheduledTask, error) {
	var task model.EdgeNodeScheduledTask
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&task).Error
	if err != nil {
		return nil, err
	}
	return &task, nil
}

// ListByNode lists scheduled tasks for a node with pagination.
func (r *EdgeNodeScheduledTaskRepository) ListByNode(ctx context.Context, nodeID string, page, pageSize int, status string) ([]model.EdgeNodeScheduledTask, int64, error) {
	query := r.db.WithContext(ctx).Model(&model.EdgeNodeScheduledTask{}).Where("node_id = ?", nodeID)
	if status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var tasks []model.EdgeNodeScheduledTask
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&tasks).Error; err != nil {
		return nil, 0, err
	}

	return tasks, total, nil
}

// ListActiveCron returns all active tasks with a non-null cron expression.
func (r *EdgeNodeScheduledTaskRepository) ListActiveCron(ctx context.Context) ([]model.EdgeNodeScheduledTask, error) {
	var tasks []model.EdgeNodeScheduledTask
	err := r.db.WithContext(ctx).
		Where("status = ? AND cron_expr IS NOT NULL AND cron_expr != ''", model.ScheduledTaskActive).
		Find(&tasks).Error
	if err != nil {
		return nil, err
	}
	return tasks, nil
}

// ListOneShotPending returns one-shot tasks that haven't been executed yet.
func (r *EdgeNodeScheduledTaskRepository) ListOneShotPending(ctx context.Context, nodeID string) ([]model.EdgeNodeScheduledTask, error) {
	var tasks []model.EdgeNodeScheduledTask
	err := r.db.WithContext(ctx).
		Where("status = ? AND (cron_expr IS NULL OR cron_expr = '') AND node_id = ? AND last_run_at IS NULL",
			model.ScheduledTaskActive, nodeID).
		Find(&tasks).Error
	if err != nil {
		return nil, err
	}
	return tasks, nil
}

// Update saves changes to a scheduled task.
func (r *EdgeNodeScheduledTaskRepository) Update(ctx context.Context, task *model.EdgeNodeScheduledTask) error {
	return r.db.WithContext(ctx).Save(task).Error
}

// Delete deletes a scheduled task by ID.
func (r *EdgeNodeScheduledTaskRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&model.EdgeNodeScheduledTask{}).Error
}

// UpdateLastRun updates the last run timestamp for a task.
func (r *EdgeNodeScheduledTaskRepository) UpdateLastRun(ctx context.Context, id string, lastRunAt interface{}) error {
	return r.db.WithContext(ctx).Model(&model.EdgeNodeScheduledTask{}).
		Where("id = ?", id).
		Update("last_run_at", lastRunAt).Error
}
