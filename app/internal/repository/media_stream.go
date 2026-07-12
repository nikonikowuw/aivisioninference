package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/niko-admin/niko-admin/internal/model"
)

// MediaStreamRepository handles media stream data persistence.
type MediaStreamRepository struct {
	db *gorm.DB
}

// NewMediaStreamRepository creates a new MediaStreamRepository.
func NewMediaStreamRepository(db *gorm.DB) *MediaStreamRepository {
	return &MediaStreamRepository{db: db}
}

// Create inserts a new media stream record.
func (r *MediaStreamRepository) Create(ctx context.Context, item *model.MediaStream) error {
	return r.db.WithContext(ctx).Create(item).Error
}

// FindByDeviceID finds media streams by device ID.
func (r *MediaStreamRepository) FindByDeviceID(ctx context.Context, deviceID string) ([]model.MediaStream, error) {
	var items []model.MediaStream
	err := r.db.WithContext(ctx).Where("device_id = ?", deviceID).Find(&items).Error
	return items, err
}

// FindActiveByDeviceAndNode finds the active stream that can share a pipeline
// for the same input device on the assigned node.
func (r *MediaStreamRepository) FindActiveByDeviceAndNode(ctx context.Context, deviceID, nodeID string) (*model.MediaStream, error) {
	var item model.MediaStream
	err := r.db.WithContext(ctx).
		Where("device_id = ? AND node_id = ? AND status = ?", deviceID, nodeID, "active").
		Order("created_at ASC").
		First(&item).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

// FindAssignedForRecovery returns active streams with a persisted node assignment.
func (r *MediaStreamRepository) FindAssignedForRecovery(ctx context.Context) ([]model.MediaStream, error) {
	var items []model.MediaStream
	err := r.db.WithContext(ctx).
		Where("node_id IS NOT NULL AND node_id <> '' AND status = ?", "active").
		Order("created_at ASC").
		Find(&items).Error
	return items, err
}

// FindByStream finds a media stream by ZLM stream identifier.
func (r *MediaStreamRepository) FindByStream(ctx context.Context, app, stream, vhost string) (*model.MediaStream, error) {
	var items []model.MediaStream
	err := r.db.WithContext(ctx).
		Where("zlm_app = ? AND zlm_stream = ? AND zlm_vhost = ?", app, stream, vhost).
		Limit(1).
		Find(&items).Error
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, nil
	}
	return &items[0], nil
}

// UpdateStatus updates the media stream status.
func (r *MediaStreamRepository) UpdateStatus(ctx context.Context, id, status string) error {
	return r.db.WithContext(ctx).Model(&model.MediaStream{}).Where("id = ?", id).
		Update("status", status).Error
}

// UpdateAssignment persists the execution node selected for a media stream.
func (r *MediaStreamRepository) UpdateAssignment(ctx context.Context, id, nodeID string) error {
	return r.db.WithContext(ctx).Model(&model.MediaStream{}).Where("id = ?", id).
		Update("node_id", nodeID).Error
}
