package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/niko-admin/niko-admin/internal/model"
)

// SmartRecordRepository handles SmartRecord persistence.
type SmartRecordRepository struct {
	db *gorm.DB
}

// NewSmartRecordRepository creates a new SmartRecordRepository.
func NewSmartRecordRepository(db *gorm.DB) *SmartRecordRepository {
	return &SmartRecordRepository{db: db}
}

// Create inserts a new smart record.
func (r *SmartRecordRepository) Create(ctx context.Context, record *model.SmartRecord) error {
	return r.db.WithContext(ctx).Create(record).Error
}

// FindByID finds a record by its ID.
func (r *SmartRecordRepository) FindByID(ctx context.Context, id string) (*model.SmartRecord, error) {
	var record model.SmartRecord
	err := r.db.WithContext(ctx).First(&record, "record_id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// List returns a paginated list of smart records based on filters.
func (r *SmartRecordRepository) List(ctx context.Context, req map[string]interface{}, page, pageSize int) ([]model.SmartRecord, int64, error) {
	var items []model.SmartRecord
	var total int64

	db := r.db.WithContext(ctx).Model(&model.SmartRecord{})

	if t, ok := req["record_type"].(string); ok && t != "" {
		db = db.Where("record_type = ?", t)
	}
	if did, ok := req["device_id"].(string); ok && did != "" {
		db = db.Where("device_id = ?", did)
	}
	if at, ok := req["alarm_type"].(string); ok && at != "" {
		// alarm_type is stored in raw_result JSON
		db = db.Where("raw_result->>'alarm_type' = ?", at)
	}
	if al, ok := req["alarm_level"].(string); ok && al != "" {
		db = db.Where("raw_result->>'alarm_level' = ?", al)
	}
	if st, ok := req["start_time"].(string); ok && st != "" {
		db = db.Where("created_at >= ?", st)
	}
	if et, ok := req["end_time"].(string); ok && et != "" {
		db = db.Where("created_at <= ?", et)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := db.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&items).Error; err != nil {
		return nil, 0, err
	}

	return items, total, nil
}
