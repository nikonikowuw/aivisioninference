package repository

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/model"
	"github.com/niko-admin/niko-admin/internal/pkg/scopes"
)

// EdgeNodeRepository handles database operations for EdgeNode model.
type EdgeNodeRepository struct {
	db *gorm.DB
}

// NewEdgeNodeRepository creates a new EdgeNodeRepository.
func NewEdgeNodeRepository(db *gorm.DB) *EdgeNodeRepository {
	return &EdgeNodeRepository{db: db}
}

// Transaction wraps operations in a database transaction.
func (r *EdgeNodeRepository) Transaction(ctx context.Context, fn func(context.Context) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(ctx)
	})
}

// Create inserts a new edge node record.
func (r *EdgeNodeRepository) Create(ctx context.Context, item *model.EdgeNode) error {
	return r.db.WithContext(ctx).Create(item).Error
}

// FindByID finds an edge node by its ID.
func (r *EdgeNodeRepository) FindByID(ctx context.Context, id string) (*model.EdgeNode, error) {
	var item model.EdgeNode
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&item).Error
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// Update saves changes to an edge node record.
func (r *EdgeNodeRepository) Update(ctx context.Context, item *model.EdgeNode) error {
	return r.db.WithContext(ctx).Save(item).Error
}

// ExistsByName 检查活动节点的名称是否已存在（排除指定 ID）
func (r *EdgeNodeRepository) ExistsByName(ctx context.Context, name, excludeID string) (bool, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&model.EdgeNode{}).Where("name = ?", name)
	if excludeID != "" {
		query = query.Where("id != ?", excludeID)
	}
	// 不使用 Unscoped()，只检查非软删除的数据
	err := query.Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// Delete soft deletes an edge node by setting its status to disabled and deleting the record.
func (r *EdgeNodeRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.EdgeNode{}).Where("id = ?", id).Update("status", model.NodeStatusDisabled).Error; err != nil {
			return err
		}
		return tx.Where("id = ?", id).Delete(&model.EdgeNode{}).Error
	})
}

// List returns a paginated list of edge nodes with optional filters.
func (r *EdgeNodeRepository) List(ctx context.Context, req dto.EdgeNodeListRequest) ([]model.EdgeNode, int64, error) {
	var items []model.EdgeNode
	var total int64

	query := r.db.WithContext(ctx).Model(&model.EdgeNode{}).Scopes(req.FilterScopes()...)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Scopes(
		scopes.Paginate(req.GetPage(), req.GetPageSize()),
		scopes.OrderBy(req.Sort, req.Order, model.EdgeNode{}.SortableFields()...),
	).Find(&items).Error
	return items, total, err
}

// UpdateHeartbeatFields selectively updates only heartbeat-related columns to prevent concurrent write conflicts.
func (r *EdgeNodeRepository) UpdateHeartbeatFields(ctx context.Context, id string, fields map[string]interface{}) error {
	// Build allowed column set for heartbeat updates
	allowed := []string{"status", "last_heartbeat", "current_load", "uptime", "engine_version",
		"hal_platform", "cpu_model", "gpu_model", "total_memory", "remark"}
	return r.db.WithContext(ctx).Model(&model.EdgeNode{}).
		Where("id = ?", id).
		Select(allowed).
		Updates(fields).Error
}

// FindByIDForUpdate finds an edge node by ID with pessimistic lock (for use in transactions).
func (r *EdgeNodeRepository) FindByIDForUpdate(ctx context.Context, id string) (*model.EdgeNode, error) {
	var item model.EdgeNode
	err := r.db.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", id).
		First(&item).Error
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// FindOnlineNodesWithAlgorithm retrieves all online and enabled nodes that have the specified algorithm installed.
func (r *EdgeNodeRepository) FindOnlineNodesWithAlgorithm(ctx context.Context, algoPackageID string) ([]model.EdgeNode, error) {
	var nodes []model.EdgeNode
	err := r.db.WithContext(ctx).
		Joins("JOIN edge_node_algorithms ON edge_nodes.id = edge_node_algorithms.node_id").
		Where("edge_nodes.status = ?", "online").
		Where("edge_nodes.enabled = ?", true).
		Where("edge_node_algorithms.algo_package_id = ?", algoPackageID).
		Where("edge_node_algorithms.status = ?", "installed").
		Find(&nodes).Error
	return nodes, err
}

// FindTimedOutNodes finds all online edge nodes whose last heartbeat is older than the cutoff time.
func (r *EdgeNodeRepository) FindTimedOutNodes(ctx context.Context, cutoff time.Time) ([]model.EdgeNode, error) {
	var items []model.EdgeNode
	err := r.db.WithContext(ctx).
		Where("status = ?", model.NodeStatusOnline).
		Where("last_heartbeat < ?", cutoff).
		Find(&items).Error
	return items, err
}

// UpdateStatusBatch updates status of multiple node IDs.
func (r *EdgeNodeRepository) UpdateStatusBatch(ctx context.Context, ids []string, status string) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Model(&model.EdgeNode{}).
		Where("id IN ?", ids).
		Update("status", status).Error
}
