package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/niko-admin/niko-admin/internal/model"
)

// AIInferenceRepository handles AI inference result persistence.
type AIInferenceRepository struct {
	db *gorm.DB
}

// NewAIInferenceRepository creates a new AIInferenceRepository.
func NewAIInferenceRepository(db *gorm.DB) *AIInferenceRepository {
	return &AIInferenceRepository{db: db}
}

// Create saves a new inference result.
func (r *AIInferenceRepository) Create(ctx context.Context, item *model.AIInferenceResult) error {
	return r.db.WithContext(ctx).Create(item).Error
}

// FindByDeviceIDAndTimeRange finds results for a device within a time range.
func (r *AIInferenceRepository) FindByDeviceIDAndTimeRange(ctx context.Context, deviceID string, start, end time.Time, page, pageSize int) ([]model.AIInferenceResult, int64, error) {
	var items []model.AIInferenceResult
	var total int64

	db := r.db.WithContext(ctx).Model(&model.AIInferenceResult{}).
		Where("device_id = ?", deviceID).
		Where("processed_at >= ? AND processed_at <= ?", start, end)

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if total > 0 {
		offset := (page - 1) * pageSize
		if err := db.Order("processed_at DESC").Offset(offset).Limit(pageSize).Find(&items).Error; err != nil {
			return nil, 0, err
		}
	}

	return items, total, nil
}
