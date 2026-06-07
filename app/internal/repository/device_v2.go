package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/niko-admin/niko-admin/internal/model"
)

// DeviceRepositoryV2 使用统一 Device 模型管理所有设备。
type DeviceRepositoryV2 struct {
	db *gorm.DB
}

func NewDeviceRepositoryV2(db *gorm.DB) *DeviceRepositoryV2 {
	return &DeviceRepositoryV2{db: db}
}

// FindGB28181NVRs 列出所有 GB28181 NVR 设备（access_type=gb28181_nvr）。
func (r *DeviceRepositoryV2) FindGB28181NVRs(ctx context.Context, keyword, status string, page, pageSize int) ([]model.Device, int64, error) {
	var items []model.Device
	var total int64
	db := r.db.WithContext(ctx).Model(&model.Device{}).Where("access_type = ?", model.DeviceAccessTypeGB28181NVR)
	if keyword != "" {
		kw := "%" + keyword + "%"
		db = db.Where("device_name LIKE ? OR gb28181_device_id LIKE ? OR manufacturer LIKE ?", kw, kw, kw)
	}
	if status != "" {
		db = db.Where("status = ?", status)
	}
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	err := db.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&items).Error
	return items, total, err
}

// FindGB28181Channels 列出指定 NVR 下的所有 GB28181 通道。
func (r *DeviceRepositoryV2) FindGB28181Channels(ctx context.Context, nvrID string, page, pageSize int) ([]model.Device, int64, error) {
	var items []model.Device
	var total int64
	db := r.db.WithContext(ctx).Model(&model.Device{}).
		Where("access_type IN (?, ?) AND parent_nvr_id = ?", model.DeviceAccessTypeGB28181, model.DeviceAccessTypeNVRChannel, nvrID)
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	err := db.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&items).Error
	return items, total, err
}

// FindGB28181NVRChannels 通过 NVR 的 GB28181 设备编码查找通道。
func (r *DeviceRepositoryV2) FindGB28181NVRChannelsByCode(ctx context.Context, deviceCode string) ([]model.Device, error) {
	var items []model.Device
	err := r.db.WithContext(ctx).
		Joins("JOIN devices nvr ON devices.parent_nvr_id = nvr.id").
		Where("nvr.gb28181_device_id = ? AND devices.access_type IN (?, ?)",
			deviceCode, model.DeviceAccessTypeGB28181, model.DeviceAccessTypeNVRChannel).
		Find(&items).Error
	return items, err
}

// UpsertByExternalKey 通过 ExternalKey 去重创建或更新。
func (r *DeviceRepositoryV2) UpsertByExternalKey(ctx context.Context, device *model.Device) error {
	return r.db.WithContext(ctx).Where(model.Device{ExternalKey: device.ExternalKey}).
		Assign(*device).FirstOrCreate(device).Error
}

// FindByID 通过 UUID 查找设备记录（预留接口，供未来扩展使用）。
func (r *DeviceRepositoryV2) FindByID(ctx context.Context, id string) (*model.Device, error) {
	var item model.Device
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&item).Error
	return &item, err
}

func (r *DeviceRepositoryV2) Update(ctx context.Context, device *model.Device) error {
	return r.db.WithContext(ctx).Save(device).Error
}

func (r *DeviceRepositoryV2) Create(ctx context.Context, device *model.Device) error {
	return r.db.WithContext(ctx).Create(device).Error
}

func (r *DeviceRepositoryV2) UpdateStatus(ctx context.Context, id, status string) error {
	now := time.Now()
	updates := map[string]interface{}{"status": status}
	if status == model.DeviceStatusOnline {
		updates["last_online_at"] = now
	} else if status == model.DeviceStatusOffline {
		updates["last_offline_at"] = now
	}
	return r.db.WithContext(ctx).Model(&model.Device{}).Where("id = ?", id).Updates(updates).Error
}
