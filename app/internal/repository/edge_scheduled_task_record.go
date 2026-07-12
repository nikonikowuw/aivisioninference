package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/model"
	"github.com/niko-admin/niko-admin/internal/pkg/scopes"
)

// EdgeScheduledTaskRecordRepository 处理计划任务执行记录的数据库操作
type EdgeScheduledTaskRecordRepository struct {
	db *gorm.DB
}

// NewEdgeScheduledTaskRecordRepository 创建新的 EdgeScheduledTaskRecordRepository
func NewEdgeScheduledTaskRecordRepository(db *gorm.DB) *EdgeScheduledTaskRecordRepository {
	return &EdgeScheduledTaskRecordRepository{db: db}
}

// FindByID 根据 ID 查询执行记录
func (r *EdgeScheduledTaskRecordRepository) FindByID(ctx context.Context, id string) (*model.EdgeScheduledTaskRecord, error) {
	var item model.EdgeScheduledTaskRecord
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&item).Error
	return &item, err
}

// Create 新增执行记录
func (r *EdgeScheduledTaskRecordRepository) Create(ctx context.Context, item *model.EdgeScheduledTaskRecord) error {
	return r.db.WithContext(ctx).Create(item).Error
}

// Update 更新执行记录
func (r *EdgeScheduledTaskRecordRepository) Update(ctx context.Context, item *model.EdgeScheduledTaskRecord) error {
	return r.db.WithContext(ctx).Save(item).Error
}

// List 分页查询执行记录
func (r *EdgeScheduledTaskRecordRepository) List(ctx context.Context, req dto.EdgeScheduledTaskRecordListRequest) ([]model.EdgeScheduledTaskRecord, int64, error) {
	var items []model.EdgeScheduledTaskRecord
	var total int64

	query := r.db.WithContext(ctx).Model(&model.EdgeScheduledTaskRecord{}).Scopes(req.FilterScopes()...)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Scopes(
		scopes.Paginate(req.GetPage(), req.GetPageSize()),
		scopes.OrderBy(req.Sort, req.Order, model.EdgeScheduledTaskRecord{}.SortableFields()...),
	).Find(&items).Error
	return items, total, err
}

// CleanupOlderThan 清理指定时间之前的执行记录
func (r *EdgeScheduledTaskRecordRepository) CleanupOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("created_at < ?", cutoff).
		Delete(&model.EdgeScheduledTaskRecord{})
	return result.RowsAffected, result.Error
}

// KeepRecentRecords 清理每个任务超过保留数量上限的旧记录（每个任务保留最近 N 条）
func (r *EdgeScheduledTaskRecordRepository) KeepRecentRecords(ctx context.Context, taskID string, keepCount int) (int64, error) {
	// 子查询找出第 keepCount 条之后的记录 ID
	subQuery := r.db.WithContext(ctx).Model(&model.EdgeScheduledTaskRecord{}).
		Where("task_id = ?", taskID).
		Order("created_at DESC").
		Select("id").
		Limit(keepCount).
		Offset(keepCount)

	result := r.db.WithContext(ctx).
		Where("task_id = ? AND id NOT IN (?)", taskID, subQuery).
		Delete(&model.EdgeScheduledTaskRecord{})
	return result.RowsAffected, result.Error
}
