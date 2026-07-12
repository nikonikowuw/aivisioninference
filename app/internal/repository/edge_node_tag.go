package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/model"
	"github.com/niko-admin/niko-admin/internal/pkg/scopes"
)

// EdgeNodeTagRepository 处理边缘节点标签的数据库操作
type EdgeNodeTagRepository struct {
	db *gorm.DB
}

// NewEdgeNodeTagRepository 创建新的 EdgeNodeTagRepository
func NewEdgeNodeTagRepository(db *gorm.DB) *EdgeNodeTagRepository {
	return &EdgeNodeTagRepository{db: db}
}

// FindByID 根据 ID 查询标签
func (r *EdgeNodeTagRepository) FindByID(ctx context.Context, id string) (*model.EdgeNodeTag, error) {
	var item model.EdgeNodeTag
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&item).Error
	return &item, err
}

// Create 新增标签
func (r *EdgeNodeTagRepository) Create(ctx context.Context, item *model.EdgeNodeTag) error {
	return r.db.WithContext(ctx).Create(item).Error
}

// Update 更新标签
func (r *EdgeNodeTagRepository) Update(ctx context.Context, item *model.EdgeNodeTag) error {
	return r.db.WithContext(ctx).Save(item).Error
}

// Delete 软删除标签
func (r *EdgeNodeTagRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&model.EdgeNodeTag{}).Error
}

// List 分页查询标签列表
func (r *EdgeNodeTagRepository) List(ctx context.Context, req dto.EdgeNodeTagListRequest) ([]model.EdgeNodeTag, int64, error) {
	var items []model.EdgeNodeTag
	var total int64

	query := r.db.WithContext(ctx).Model(&model.EdgeNodeTag{}).Scopes(req.FilterScopes()...)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Scopes(
		scopes.Paginate(req.GetPage(), req.GetPageSize()),
		scopes.OrderBy(req.Sort, req.Order, model.EdgeNodeTag{}.SortableFields()...),
	).Find(&items).Error
	return items, total, err
}

// ListAll 返回全部标签（供下拉选择使用）
func (r *EdgeNodeTagRepository) ListAll(ctx context.Context) ([]model.EdgeNodeTag, error) {
	var items []model.EdgeNodeTag
	err := r.db.WithContext(ctx).Order("name ASC").Find(&items).Error
	return items, err
}

// FindNodesByTagID 查询标签关联的所有节点 ID
func (r *EdgeNodeTagRepository) FindNodesByTagID(ctx context.Context, tagID string) ([]string, error) {
	var nodeIDs []string
	err := r.db.WithContext(ctx).Model(&model.EdgeNodeTagRelation{}).
		Where("edge_node_tag_id = ?", tagID).
		Pluck("edge_node_id", &nodeIDs).Error
	return nodeIDs, err
}

// ReplaceNodeTags 替换节点的标签关联（全量替换）
func (r *EdgeNodeTagRepository) ReplaceNodeTags(ctx context.Context, nodeID string, tagIDs []string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 删除原有关联
		if err := tx.Where("edge_node_id = ?", nodeID).Delete(&model.EdgeNodeTagRelation{}).Error; err != nil {
			return err
		}
		// 新增关联
		for _, tagID := range tagIDs {
			rel := model.EdgeNodeTagRelation{
				EdgeNodeID:    nodeID,
				EdgeNodeTagID: tagID,
			}
			if err := tx.Create(&rel).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
