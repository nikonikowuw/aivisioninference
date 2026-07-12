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

// DB returns the underlying GORM DB instance for manual transaction management.
func (r *EdgeNodeRepository) DB(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx)
}

// NewEdgeNodeRepository creates a new EdgeNodeRepository.
func NewEdgeNodeRepository(db *gorm.DB) *EdgeNodeRepository {
	return &EdgeNodeRepository{db: db}
}

// WithTx returns a repository bound to the provided transaction.
func (r *EdgeNodeRepository) WithTx(tx *gorm.DB) *EdgeNodeRepository {
	return &EdgeNodeRepository{db: tx}
}

// Transaction wraps operations in a database transaction.
// The callback receives an *EdgeNodeRepository bound to the transaction,
// so all read/write operations on it share the same transactional connection.
func (r *EdgeNodeRepository) Transaction(ctx context.Context, fn func(txRepo *EdgeNodeRepository) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(r.WithTx(tx))
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
		"hal_platform", "cpu_model", "gpu_model", "total_memory", "embedding_capacity", "remark"}
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

// ListMediaSchedulableForUpdate locks online and enabled media nodes in a stable order.
func (r *EdgeNodeRepository) ListMediaSchedulableForUpdate(ctx context.Context) ([]model.EdgeNode, error) {
	var items []model.EdgeNode
	err := r.db.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("status = ? AND enabled = ?", model.NodeStatusOnline, true).
		Order("id ASC").
		Find(&items).Error
	return items, err
}

// edgeNodeAlgoQueryOptions controls optional filters for the shared edge-node + edge-node-algorithm join query.
type edgeNodeAlgoQueryOptions struct {
	runtimeStatus       []string // empty = no filter; single = equality; multiple = IN
	supportsEmbedding   *bool
	supportsFaceLibrary *bool
	order               string
}

// findNodesByAlgoJoin is the shared query builder for the four Find*NodesByAlgorithm methods.
// It joins edge_nodes with edge_node_algorithms and applies the common base filters
// (online, enabled, installed, algo_package_id) plus any optional filters from opts.
func (r *EdgeNodeRepository) findNodesByAlgoJoin(ctx context.Context, algoPackageID string, opts edgeNodeAlgoQueryOptions) ([]model.EdgeNode, error) {
	var nodes []model.EdgeNode
	query := r.db.WithContext(ctx).
		Joins("JOIN edge_node_algorithms ON edge_nodes.id = edge_node_algorithms.node_id").
		Where("edge_nodes.status = ?", model.NodeStatusOnline).
		Where("edge_nodes.enabled = ?", true).
		Where("edge_node_algorithms.algo_package_id = ?", algoPackageID).
		Where("edge_node_algorithms.status = ?", model.AlgoDeployInstalled)
	if len(opts.runtimeStatus) == 1 {
		query = query.Where("edge_node_algorithms.runtime_status = ?", opts.runtimeStatus[0])
	} else if len(opts.runtimeStatus) > 1 {
		query = query.Where("edge_node_algorithms.runtime_status IN ?", opts.runtimeStatus)
	}
	if opts.supportsEmbedding != nil {
		query = query.Where("edge_node_algorithms.supports_embedding = ?", *opts.supportsEmbedding)
	}
	if opts.supportsFaceLibrary != nil {
		query = query.Where("edge_node_algorithms.supports_face_library = ?", *opts.supportsFaceLibrary)
	}
	if opts.order != "" {
		query = query.Order(opts.order)
	}
	err := query.Find(&nodes).Error
	return nodes, err
}

// FindOnlineNodesWithAlgorithm retrieves all online and enabled nodes that have the specified algorithm installed.
func (r *EdgeNodeRepository) FindOnlineNodesWithAlgorithm(ctx context.Context, algoPackageID string) ([]model.EdgeNode, error) {
	return r.findNodesByAlgoJoin(ctx, algoPackageID, edgeNodeAlgoQueryOptions{})
}

// FindReadyEmbeddingNodesByAlgorithm returns online nodes that can immediately serve embedding extraction.
func (r *EdgeNodeRepository) FindReadyEmbeddingNodesByAlgorithm(ctx context.Context, algoPackageID string) ([]model.EdgeNode, error) {
	supportsEmbedding := true
	return r.findNodesByAlgoJoin(ctx, algoPackageID, edgeNodeAlgoQueryOptions{
		runtimeStatus:     []string{model.AlgoRuntimeReady},
		supportsEmbedding: &supportsEmbedding,
		order:             "edge_nodes.current_load ASC, edge_nodes.updated_at DESC",
	})
}

// FindInstalledEmbeddingNodesByAlgorithm returns online nodes that have the package installed but are not yet ready.
func (r *EdgeNodeRepository) FindInstalledEmbeddingNodesByAlgorithm(ctx context.Context, algoPackageID string) ([]model.EdgeNode, error) {
	supportsEmbedding := true
	return r.findNodesByAlgoJoin(ctx, algoPackageID, edgeNodeAlgoQueryOptions{
		runtimeStatus:     []string{model.AlgoRuntimeInstalled, model.AlgoRuntimeFailed, model.AlgoRuntimeUnknown},
		supportsEmbedding: &supportsEmbedding,
		order:             "edge_nodes.current_load ASC, edge_nodes.updated_at DESC",
	})
}

// FindReadyFaceLibraryNodesByAlgorithm returns nodes that should receive face library snapshots.
func (r *EdgeNodeRepository) FindReadyFaceLibraryNodesByAlgorithm(ctx context.Context, algoPackageID string) ([]model.EdgeNode, error) {
	supportsFaceLibrary := true
	return r.findNodesByAlgoJoin(ctx, algoPackageID, edgeNodeAlgoQueryOptions{
		runtimeStatus:       []string{model.AlgoRuntimeReady},
		supportsFaceLibrary: &supportsFaceLibrary,
		order:               "edge_nodes.updated_at DESC",
	})
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

// MarkOffline transitions a non-disabled node to offline once.
func (r *EdgeNodeRepository) MarkOffline(ctx context.Context, id string) (bool, error) {
	result := r.db.WithContext(ctx).Model(&model.EdgeNode{}).
		Where("id = ?", id).
		Where("status NOT IN ?", []string{model.NodeStatusOffline, model.NodeStatusDisabled}).
		Update("status", model.NodeStatusOffline)
	return result.RowsAffected > 0, result.Error
}

// MarkOfflineIfTimedOut transitions an online node only when its persisted heartbeat is stale.
func (r *EdgeNodeRepository) MarkOfflineIfTimedOut(ctx context.Context, id string, cutoff time.Time) (bool, error) {
	result := r.db.WithContext(ctx).Model(&model.EdgeNode{}).
		Where("id = ?", id).
		Where("status = ?", model.NodeStatusOnline).
		Where("last_heartbeat < ?", cutoff).
		Update("status", model.NodeStatusOffline)
	return result.RowsAffected > 0, result.Error
}
