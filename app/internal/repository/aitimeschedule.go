package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/model"
	"github.com/niko-admin/niko-admin/internal/pkg/scopes"
)

// AITimeScheduleRepository 处理 AITimeSchedule 的数据库操作
type AITimeScheduleRepository struct {
	db *gorm.DB
}

// NewAITimeScheduleRepository 创建新的 AITimeScheduleRepository
func NewAITimeScheduleRepository(db *gorm.DB) *AITimeScheduleRepository {
	return &AITimeScheduleRepository{db: db}
}

// FindByID 根据 ID 查询时间配置
func (r *AITimeScheduleRepository) FindByID(ctx context.Context, id string) (*model.AITimeSchedule, error) {
	var item model.AITimeSchedule
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&item).Error
	return &item, err
}

// Create 新增时间配置
func (r *AITimeScheduleRepository) Create(ctx context.Context, item *model.AITimeSchedule) error {
	return r.db.WithContext(ctx).Create(item).Error
}

// Update 更新时间配置
func (r *AITimeScheduleRepository) Update(ctx context.Context, item *model.AITimeSchedule) error {
	return r.db.WithContext(ctx).Save(item).Error
}

// Delete 软删除时间配置
func (r *AITimeScheduleRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&model.AITimeSchedule{}).Error
}

// List 分页查询时间配置
func (r *AITimeScheduleRepository) List(ctx context.Context, req dto.AITimeScheduleListRequest) ([]model.AITimeSchedule, int64, error) {
	var items []model.AITimeSchedule
	var total int64

	query := r.db.WithContext(ctx).Model(&model.AITimeSchedule{}).Scopes(req.FilterScopes()...)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Scopes(
		scopes.Paginate(req.GetPage(), req.GetPageSize()),
		scopes.OrderBy(req.Sort, req.Order, model.AITimeSchedule{}.SortableFields()...),
	).Find(&items).Error
	return items, total, err
}

// ListAll 返回全部时间配置（供下拉选择使用）
func (r *AITimeScheduleRepository) ListAll(ctx context.Context) ([]model.AITimeSchedule, error) {
	var items []model.AITimeSchedule
	err := r.db.WithContext(ctx).Order("name ASC").Find(&items).Error
	return items, err
}
