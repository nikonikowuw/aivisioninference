package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/niko-admin/niko-admin/internal/model"
)

// GB28181DeviceRepository handles GB28181 device data persistence.
type GB28181DeviceRepository struct {
	db *gorm.DB
}

// NewGB28181DeviceRepository creates a new GB28181DeviceRepository.
func NewGB28181DeviceRepository(db *gorm.DB) *GB28181DeviceRepository {
	return &GB28181DeviceRepository{db: db}
}

// Create inserts a new GB28181 device record.
func (r *GB28181DeviceRepository) Create(ctx context.Context, item *model.GB28181Device) error {
	return r.db.WithContext(ctx).Create(item).Error
}

// FindByDeviceCode finds a device by its GB28181 device code.
func (r *GB28181DeviceRepository) FindByDeviceCode(ctx context.Context, deviceCode string) (*model.GB28181Device, error) {
	var item model.GB28181Device
	err := r.db.WithContext(ctx).Where("device_code = ?", deviceCode).First(&item).Error
	return &item, err
}

// FindByDeviceCodes finds devices by multiple device codes.
func (r *GB28181DeviceRepository) FindByDeviceCodes(ctx context.Context, deviceCodes []string) ([]model.GB28181Device, error) {
	if len(deviceCodes) == 0 {
		return nil, nil
	}
	var items []model.GB28181Device
	err := r.db.WithContext(ctx).Where("device_code IN ?", deviceCodes).Find(&items).Error
	return items, err
}

// UpdateStatus updates the device status.
func (r *GB28181DeviceRepository) UpdateStatus(ctx context.Context, id, status string) error {
	return r.db.WithContext(ctx).Model(&model.GB28181Device{}).Where("id = ?", id).
		Update("status", status).Error
}

// FindByID finds a device by its internal UUID.
func (r *GB28181DeviceRepository) FindByID(ctx context.Context, id string) (*model.GB28181Device, error) {
	var item model.GB28181Device
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&item).Error
	return &item, err
}

// List returns a paginated list of GB28181 devices based on filters.
func (r *GB28181DeviceRepository) List(ctx context.Context, keyword string, status string, page, pageSize int) ([]model.GB28181Device, int64, error) {
	var items []model.GB28181Device
	var total int64

	db := r.db.WithContext(ctx).Model(&model.GB28181Device{})

	if keyword != "" {
		kw := "%" + keyword + "%"
		db = db.Where("device_code LIKE ? OR manufacturer LIKE ? OR model LIKE ?", kw, kw, kw)
	}
	if status != "" {
		db = db.Where("status = ?", status)
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

// Update updates the device information.
func (r *GB28181DeviceRepository) Update(ctx context.Context, id string, updates map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&model.GB28181Device{}).Where("id = ?", id).Updates(updates).Error
}

// Delete deletes the device.
func (r *GB28181DeviceRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&model.GB28181Device{}).Error
}
