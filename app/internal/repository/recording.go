package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/niko-admin/niko-admin/internal/model"
)

// RecordingRepository handles media recording data persistence.
type RecordingRepository struct {
	db *gorm.DB
}

// NewRecordingRepository creates a new RecordingRepository.
func NewRecordingRepository(db *gorm.DB) *RecordingRepository {
	return &RecordingRepository{db: db}
}

// Create inserts a new recording record.
func (r *RecordingRepository) Create(ctx context.Context, item *model.MediaRecording) error {
	return r.db.WithContext(ctx).Create(item).Error
}

// FindByDeviceIDAndTimeRange finds recordings by device ID and time range.
func (r *RecordingRepository) FindByDeviceIDAndTimeRange(
	ctx context.Context, deviceID string, start, end time.Time, page, pageSize int,
) ([]model.MediaRecording, int64, error) {
	var items []model.MediaRecording
	var total int64

	db := r.db.WithContext(ctx).Model(&model.MediaRecording{}).
		Where("device_id = ?", deviceID).
		Where("start_time >= ? AND end_time <= ?", start, end)

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if total == 0 {
		return items, 0, nil
	}

	offset := (page - 1) * pageSize
	if err := db.Order("start_time DESC").Offset(offset).Limit(pageSize).Find(&items).Error; err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

// FindByID finds a recording by ID.
func (r *RecordingRepository) FindByID(ctx context.Context, id string) (*model.MediaRecording, error) {
	var item model.MediaRecording
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&item).Error
	return &item, err
}

// DeleteOlderThan deletes recordings older than the specified duration.
func (r *RecordingRepository) DeleteOlderThan(ctx context.Context, maxAge time.Duration) (int64, error) {
	cutoff := time.Now().Add(-maxAge)
	result := r.db.WithContext(ctx).Where("start_time < ?", cutoff).Delete(&model.MediaRecording{})
	return result.RowsAffected, result.Error
}

// DeleteByID deletes a recording by ID.
func (r *RecordingRepository) DeleteByID(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&model.MediaRecording{}, "id = ?", id).Error
}
