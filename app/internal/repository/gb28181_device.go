package repository

import (
	"context"
	"time"

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

// UpdateHeartbeat updates the last heartbeat timestamp for a device.
func (r *GB28181DeviceRepository) UpdateHeartbeat(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Model(&model.GB28181Device{}).Where("id = ?", id).
		Update("last_heartbeat_at", time.Now()).Error
}

// UpdateStatus updates the device status.
func (r *GB28181DeviceRepository) UpdateStatus(ctx context.Context, id, status string) error {
	return r.db.WithContext(ctx).Model(&model.GB28181Device{}).Where("id = ?", id).
		Update("status", status).Error
}

// FindOfflineDevices finds devices that haven't sent heartbeat within the given timeout.
func (r *GB28181DeviceRepository) FindOfflineDevices(ctx context.Context, heartbeatTimeout time.Duration) ([]model.GB28181Device, error) {
	var items []model.GB28181Device
	cutoff := time.Now().Add(-heartbeatTimeout)
	err := r.db.WithContext(ctx).
		Where("status = ? AND (last_heartbeat_at IS NULL OR last_heartbeat_at < ?)", model.GB28181StatusOnline, cutoff).
		Find(&items).Error
	return items, err
}
