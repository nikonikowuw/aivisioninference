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

// FindByStream finds a media stream by ZLM stream identifier.
func (r *MediaStreamRepository) FindByStream(ctx context.Context, app, stream, vhost string) (*model.MediaStream, error) {
	var item model.MediaStream
	err := r.db.WithContext(ctx).Where("zlm_app = ? AND zlm_stream = ? AND zlm_vhost = ?", app, stream, vhost).First(&item).Error
	return &item, err
}

// UpdateStatus updates the media stream status.
func (r *MediaStreamRepository) UpdateStatus(ctx context.Context, id, status string) error {
	return r.db.WithContext(ctx).Model(&model.MediaStream{}).Where("id = ?", id).
		Update("status", status).Error
}
