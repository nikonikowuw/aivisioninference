package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/model"
	"github.com/niko-admin/niko-admin/internal/pkg/scopes"
)

// EdgeScheduledTaskRepository 处理计划任务的数据库操作
type EdgeScheduledTaskRepository struct {
	db *gorm.DB
}

// NewEdgeScheduledTaskRepository 创建新的 EdgeScheduledTaskRepository
func NewEdgeScheduledTaskRepository(db *gorm.DB) *EdgeScheduledTaskRepository {
	return &EdgeScheduledTaskRepository{db: db}
}

// FindByID 根据 ID 查询计划任务
func (r *EdgeScheduledTaskRepository) FindByID(ctx context.Context, id string) (*model.EdgeScheduledTask, error) {
	var item model.EdgeScheduledTask
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&item).Error
	return &item, err
}

// Create 新增计划任务
func (r *EdgeScheduledTaskRepository) Create(ctx context.Context, item *model.EdgeScheduledTask) error {
	return r.db.WithContext(ctx).Create(item).Error
}

// Update 更新计划任务
func (r *EdgeScheduledTaskRepository) Update(ctx context.Context, item *model.EdgeScheduledTask) error {
	return r.db.WithContext(ctx).Save(item).Error
}

// Delete 软删除计划任务
func (r *EdgeScheduledTaskRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&model.EdgeScheduledTask{}).Error
}

// List 分页查询计划任务列表
func (r *EdgeScheduledTaskRepository) List(ctx context.Context, req dto.EdgeScheduledTaskListRequest) ([]model.EdgeScheduledTask, int64, error) {
	var items []model.EdgeScheduledTask
	var total int64

	query := r.db.WithContext(ctx).Model(&model.EdgeScheduledTask{}).Scopes(req.FilterScopes()...)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Scopes(
		scopes.Paginate(req.GetPage(), req.GetPageSize()),
		scopes.OrderBy(req.Sort, req.Order, model.EdgeScheduledTask{}.SortableFields()...),
	).Find(&items).Error
	return items, total, err
}

// ListEnabled 查询所有启用的计划任务
func (r *EdgeScheduledTaskRepository) ListEnabled(ctx context.Context) ([]model.EdgeScheduledTask, error) {
	var items []model.EdgeScheduledTask
	err := r.db.WithContext(ctx).Where("enabled = ?", true).Find(&items).Error
	return items, err
}
