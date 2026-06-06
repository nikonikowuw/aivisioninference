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
