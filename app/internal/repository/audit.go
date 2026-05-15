package repository

import (
	"context"
	"time"

	"github.com/niko-admin/niko-admin/internal/model"
	"gorm.io/gorm"
)

// AuditLogFilters holds typed filter parameters for audit log queries.
type AuditLogFilters struct {
	UserID       string
	Action       string
	ResourceType string
	StartTime    *time.Time
	EndTime      *time.Time
}

// AuditRepository handles database operations for AuditLog model.
type AuditRepository struct {
	db *gorm.DB
}

// NewAuditRepository creates a new AuditRepository.
func NewAuditRepository(db *gorm.DB) *AuditRepository {
	return &AuditRepository{db: db}
}

// Create inserts a new audit log record.
func (r *AuditRepository) Create(ctx context.Context, log *model.AuditLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

// List returns a paginated list of audit logs with optional filters.
func (r *AuditRepository) List(ctx context.Context, page, pageSize int, filters map[string]interface{}) ([]model.AuditLog, int64, error) {
	var logs []model.AuditLog
	var total int64

	query := r.db.WithContext(ctx).Model(&model.AuditLog{})
	for k, v := range filters {
		query = query.Where(k, v)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&logs).Error
	return logs, total, err
}

// ListWithFilters returns a paginated list of audit logs with typed filters.
func (r *AuditRepository) ListWithFilters(ctx context.Context, page, pageSize int, filters AuditLogFilters) ([]model.AuditLog, int64, error) {
	var logs []model.AuditLog
	var total int64

	query := r.db.WithContext(ctx).Model(&model.AuditLog{})
	if filters.UserID != "" {
		query = query.Where("user_id = ?", filters.UserID)
	}
	if filters.Action != "" {
		query = query.Where("action = ?", filters.Action)
	}
	if filters.ResourceType != "" {
		query = query.Where("resource_type = ?", filters.ResourceType)
	}
	if filters.StartTime != nil {
		query = query.Where("created_at >= ?", *filters.StartTime)
	}
	if filters.EndTime != nil {
		query = query.Where("created_at <= ?", *filters.EndTime)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&logs).Error
	return logs, total, err
}
