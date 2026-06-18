package repository

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/model"
)

// EdgeNodeAlgorithmRepository handles database operations for EdgeNodeAlgorithm model.
type EdgeNodeAlgorithmRepository struct {
	db *gorm.DB
}

// NewEdgeNodeAlgorithmRepository creates a new EdgeNodeAlgorithmRepository.
func NewEdgeNodeAlgorithmRepository(db *gorm.DB) *EdgeNodeAlgorithmRepository {
	return &EdgeNodeAlgorithmRepository{db: db}
}

// Delete permanently deletes a deployment record (hard delete, as the unique
// index conflicts with soft-deleted records on re-deployment).
func (r *EdgeNodeAlgorithmRepository) Delete(ctx context.Context, nodeID string, algoPackageID string) error {
	return r.db.WithContext(ctx).
		Unscoped().
		Where("node_id = ? AND algo_package_id = ?", nodeID, algoPackageID).
		Delete(&model.EdgeNodeAlgorithm{}).Error
}

// Create inserts a new edge node algorithm record.
func (r *EdgeNodeAlgorithmRepository) Create(ctx context.Context, item *model.EdgeNodeAlgorithm) error {
	return r.db.WithContext(ctx).Create(item).Error
}

// FindByNodeAndAlgo finds a record by node ID and algorithm package ID.
func (r *EdgeNodeAlgorithmRepository) FindByNodeAndAlgo(ctx context.Context, nodeID string, algoPackageID string) (*model.EdgeNodeAlgorithm, error) {
	var item model.EdgeNodeAlgorithm
	err := r.db.WithContext(ctx).
		Where("node_id = ? AND algo_package_id = ?", nodeID, algoPackageID).
		Preload("AlgoPackage").
		First(&item).Error
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// ListByNode returns all algorithms deployed on a specific node.
func (r *EdgeNodeAlgorithmRepository) ListByNode(ctx context.Context, nodeID string) ([]model.EdgeNodeAlgorithm, error) {
	var items []model.EdgeNodeAlgorithm
	err := r.db.WithContext(ctx).
		Where("node_id = ?", nodeID).
		Preload("AlgoPackage").
		Find(&items).Error
	return items, err
}

// UpdateStatus updates the status of a deployment record.
func (r *EdgeNodeAlgorithmRepository) UpdateStatus(ctx context.Context, nodeID string, algoPackageID string, status string) error {
	return r.db.WithContext(ctx).
		Model(&model.EdgeNodeAlgorithm{}).
		Where("node_id = ? AND algo_package_id = ?", nodeID, algoPackageID).
		Update("status", status).Error
}

// UpdateInstallInfo updates the path and deployment timestamp.
func (r *EdgeNodeAlgorithmRepository) UpdateInstallInfo(ctx context.Context, nodeID string, algoPackageID string, installPath string, deployedAt time.Time) error {
	return r.db.WithContext(ctx).
		Model(&model.EdgeNodeAlgorithm{}).
		Where("node_id = ? AND algo_package_id = ?", nodeID, algoPackageID).
		Updates(map[string]interface{}{
			"status":        model.AlgoDeployInstalled,
			"install_path":  installPath,
			"deployed_at":   deployedAt,
			"error_message": "",
		}).Error
}

// ListPendingByNode returns all pending algorithm deployments for a node.
func (r *EdgeNodeAlgorithmRepository) ListPendingByNode(ctx context.Context, nodeID string) ([]model.EdgeNodeAlgorithm, error) {
	var items []model.EdgeNodeAlgorithm
	err := r.db.WithContext(ctx).
		Where("node_id = ? AND status = ?", nodeID, model.AlgoDeployPending).
		Preload("AlgoPackage").
		Find(&items).Error
	return items, err
}

// SyncInstalled synchronizes the installed algorithms status reported by the node.
func (r *EdgeNodeAlgorithmRepository) SyncInstalled(ctx context.Context, nodeID string, installed []dto.InstalledAlgorithmInfo) error {
	if len(installed) == 0 {
		return nil
	}

	packageIDs := make([]string, 0, len(installed))
	latestByPackageID := make(map[string]dto.InstalledAlgorithmInfo, len(installed))
	for _, info := range installed {
		if info.AlgoPackageID == "" {
			continue
		}
		if _, exists := latestByPackageID[info.AlgoPackageID]; !exists {
			packageIDs = append(packageIDs, info.AlgoPackageID)
		}
		latestByPackageID[info.AlgoPackageID] = info
	}
	if len(packageIDs) == 0 {
		return nil
	}

	var existing []model.EdgeNodeAlgorithm
	if err := r.db.WithContext(ctx).
		Where("node_id = ? AND algo_package_id IN ?", nodeID, packageIDs).
		Find(&existing).Error; err != nil {
		return err
	}

	existingByPackageID := make(map[string]model.EdgeNodeAlgorithm, len(existing))
	for _, item := range existing {
		existingByPackageID[item.AlgoPackageID] = item
	}

	now := time.Now()
	updates := make([]model.EdgeNodeAlgorithm, 0, len(existing))
	for _, algoPackageID := range packageIDs {
		info := latestByPackageID[algoPackageID]
		item, ok := existingByPackageID[algoPackageID]
		if !ok {
			continue // Skip auto-creation.
		}

		switch info.Status {
		case "installed":
			if item.Status != model.AlgoDeployInstalled {
				item.Status = model.AlgoDeployInstalled
				item.InstallPath = info.InstallPath
				item.DeployedAt = &now
				item.ErrorMessage = ""
			}
			if info.RuntimeStatus != "" {
				item.RuntimeStatus = info.RuntimeStatus
			} else if item.RuntimeStatus == "" {
				item.RuntimeStatus = model.AlgoRuntimeInstalled
			}
		case "failed":
			if item.Status != model.AlgoDeployFailed {
				item.Status = model.AlgoDeployFailed
				item.ErrorMessage = "引擎端安装失败"
			}
			if info.RuntimeStatus != "" {
				item.RuntimeStatus = info.RuntimeStatus
			} else {
				item.RuntimeStatus = model.AlgoRuntimeFailed
			}
		default:
			continue
		}

		item.SupportsEmbedding = info.SupportsEmbedding
		item.SupportsFaceLibrary = info.SupportsFaceLibrary
		if info.EmbeddingCapacity > 0 {
			item.EmbeddingCapacity = info.EmbeddingCapacity
		}
		item.UpdatedAt = now
		updates = append(updates, item)
	}
	if len(updates) == 0 {
		return nil
	}

	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "node_id"},
				{Name: "algo_package_id"},
			},
			DoUpdates: clause.AssignmentColumns([]string{
				"status",
				"install_path",
				"runtime_status",
				"supports_embedding",
				"supports_face_library",
				"embedding_capacity",
				"deployed_at",
				"error_message",
				"updated_at",
			}),
		}).
		CreateInBatches(updates, 100).Error
}

// FindFailedWithRetries finds failed algorithm deployments with less than 3 retries.
func (r *EdgeNodeAlgorithmRepository) FindFailedWithRetries(ctx context.Context) ([]model.EdgeNodeAlgorithm, error) {
	var deployments []model.EdgeNodeAlgorithm
	err := r.db.WithContext(ctx).
		Where("status = ? AND retry_count < ?", model.AlgoDeployFailed, 3). // MaxRetryCount = 3
		Preload("AlgoPackage").
		Find(&deployments).Error
	return deployments, err
}

// UpdateForRetry updates fields for a retry attempt.
func (r *EdgeNodeAlgorithmRepository) UpdateForRetry(ctx context.Context, nodeID string, algoPackageID string, updates map[string]interface{}) error {
	return r.db.WithContext(ctx).
		Model(&model.EdgeNodeAlgorithm{}).
		Where("node_id = ? AND algo_package_id = ?", nodeID, algoPackageID).
		Updates(updates).Error
}

// ResetDeployment resets a failed deployment back to pending, clearing error and retry fields.
func (r *EdgeNodeAlgorithmRepository) ResetDeployment(ctx context.Context, nodeID string, algoPackageID string) error {
	return r.db.WithContext(ctx).
		Model(&model.EdgeNodeAlgorithm{}).
		Where("node_id = ? AND algo_package_id = ?", nodeID, algoPackageID).
		Updates(map[string]interface{}{
			"status":        model.AlgoDeployPending,
			"error_message": "",
			"install_path":  "",
			"retry_count":   0,
			"last_retry_at": nil,
		}).Error
}

// RestoreDeleted finds a soft-deleted record and restores it to pending.
// Returns the restored record, or ErrRecordNotFound if no soft-deleted record exists.
func (r *EdgeNodeAlgorithmRepository) RestoreDeleted(ctx context.Context, nodeID string, algoPackageID string) (*model.EdgeNodeAlgorithm, error) {
	var item model.EdgeNodeAlgorithm
	err := r.db.WithContext(ctx).
		Unscoped().
		Where("node_id = ? AND algo_package_id = ?", nodeID, algoPackageID).
		First(&item).Error
	if err != nil {
		return nil, err
	}
	if !item.DeletedAt.Valid {
		// Not soft-deleted
		return &item, nil
	}
	// Restore: clear deleted_at and reset status
	item.DeletedAt = gorm.DeletedAt{Time: time.Time{}, Valid: false}
	item.Status = model.AlgoDeployPending
	item.ErrorMessage = ""
	item.InstallPath = ""
	item.RetryCount = 0
	if err := r.db.WithContext(ctx).Unscoped().Save(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}
