package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/niko-admin/niko-admin/internal/model"
)

// DeviceSipConfigRepository 管理 device_sip_configs 表。
type DeviceSipConfigRepository struct {
	db *gorm.DB
}

func NewDeviceSipConfigRepository(db *gorm.DB) *DeviceSipConfigRepository {
	return &DeviceSipConfigRepository{db: db}
}

func (r *DeviceSipConfigRepository) Create(ctx context.Context, cfg *model.DeviceSipConfig) error {
	return r.db.WithContext(ctx).Create(cfg).Error
}

func (r *DeviceSipConfigRepository) FindByID(ctx context.Context, id string) (*model.DeviceSipConfig, error) {
	var cfg model.DeviceSipConfig
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&cfg).Error
	return &cfg, err
}

func (r *DeviceSipConfigRepository) FindByDeviceID(ctx context.Context, deviceID string) (*model.DeviceSipConfig, error) {
	var cfg model.DeviceSipConfig
	err := r.db.WithContext(ctx).Where("device_id = ?", deviceID).First(&cfg).Error
	return &cfg, err
}

func (r *DeviceSipConfigRepository) FindByDeviceCode(ctx context.Context, deviceCode string) (*model.DeviceSipConfig, error) {
	var cfg model.DeviceSipConfig
	err := r.db.WithContext(ctx).Where("device_code = ?", deviceCode).First(&cfg).Error
	return &cfg, err
}

func (r *DeviceSipConfigRepository) FindByDeviceCodes(ctx context.Context, deviceCodes []string) ([]model.DeviceSipConfig, error) {
	var cfgs []model.DeviceSipConfig
	err := r.db.WithContext(ctx).Where("device_code IN ?", deviceCodes).Find(&cfgs).Error
	return cfgs, err
}

func (r *DeviceSipConfigRepository) Update(ctx context.Context, id string, updates map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&model.DeviceSipConfig{}).Where("id = ?", id).Updates(updates).Error
}

func (r *DeviceSipConfigRepository) UpdateHeartbeat(ctx context.Context, deviceCode string) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&model.DeviceSipConfig{}).Where("device_code = ?", deviceCode).
		Updates(map[string]interface{}{"last_heartbeat_at": now}).Error
}

func (r *DeviceSipConfigRepository) UpdateRegisterAddress(ctx context.Context, deviceCode, addr string, port int) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&model.DeviceSipConfig{}).Where("device_code = ?", deviceCode).
		Updates(map[string]interface{}{
			"register_address":  addr,
			"register_port":     port,
			"last_register_at":  now,
		}).Error
}


func (r *DeviceSipConfigRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&model.DeviceSipConfig{}).Error
}
