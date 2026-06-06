package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/niko-admin/niko-admin/internal/model"
)

// DiscoveredDeviceRepository 处理设备发现结果持久化
type DiscoveredDeviceRepository struct {
	db *gorm.DB
}

// NewDiscoveredDeviceRepository 创建一个新的 DiscoveredDeviceRepository 实例
func NewDiscoveredDeviceRepository(db *gorm.DB) *DiscoveredDeviceRepository {
	return &DiscoveredDeviceRepository{db: db}
}

// List 返回分页的发现结果
func (r *DiscoveredDeviceRepository) List(ctx context.Context, source, status, keyword string, page, pageSize int) ([]model.DiscoveredDevice, int64, error) {
	var items []model.DiscoveredDevice
	var total int64

	db := r.db.WithContext(ctx).Model(&model.DiscoveredDevice{})

	if source != "" {
		db = db.Where("source = ?", source)
	}
	if status != "" {
		db = db.Where("status = ?", status)
	}
	if keyword != "" {
		db = db.Where("(device_name ILIKE ? OR device_ip ILIKE ? OR manufacturer ILIKE ?)", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err := db.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&items).Error
	return items, total, err
}

// Create 插入记录
func (r *DiscoveredDeviceRepository) Create(ctx context.Context, item *model.DiscoveredDevice) error {
	return r.db.WithContext(ctx).Create(item).Error
}

// Upsert 根据 MAC 或 IP+Source 去重更新
func (r *DiscoveredDeviceRepository) Upsert(ctx context.Context, item *model.DiscoveredDevice) error {
	// 优先根据 MAC 去重
	if item.DeviceMAC != "" {
		return r.db.WithContext(ctx).Where("device_mac = ?", item.DeviceMAC).
			Assign(model.DiscoveredDevice{
				DeviceIP:        item.DeviceIP,
				DeviceName:      item.DeviceName,
				Manufacturer:    item.Manufacturer,
				Model:           item.Model,
				FirmwareVersion: item.FirmwareVersion,
				AccessURL:       item.AccessURL,
				ExtraInfo:       item.ExtraInfo,
			}).
			FirstOrCreate(item).Error
	}
	
	// 备选根据 IP + Source 去重（适用于 MAC 无法获取的情况）
	return r.db.WithContext(ctx).Where("device_ip = ? AND source = ?", item.DeviceIP, item.Source).
		Assign(model.DiscoveredDevice{
			DeviceName:      item.DeviceName,
			Manufacturer:    item.Manufacturer,
			Model:           item.Model,
			FirmwareVersion: item.FirmwareVersion,
			AccessURL:       item.AccessURL,
			ExtraInfo:       item.ExtraInfo,
		}).
		FirstOrCreate(item).Error
}

// BatchUpdateStatus 批量更新状态
func (r *DiscoveredDeviceRepository) BatchUpdateStatus(ctx context.Context, ids []string, status string) error {
	updates := map[string]interface{}{"status": status}
	if status == model.StatusIgnored {
		now := time.Now()
		updates["ignored_at"] = &now
	}
	return r.db.WithContext(ctx).Model(&model.DiscoveredDevice{}).Where("id IN ?", ids).Updates(updates).Error
}

// FindByID 根据 ID 查询
func (r *DiscoveredDeviceRepository) FindByID(ctx context.Context, id string) (*model.DiscoveredDevice, error) {
	var item model.DiscoveredDevice
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&item).Error
	return &item, err
}

// MarkImported 标记已导入
func (r *DiscoveredDeviceRepository) MarkImported(ctx context.Context, id, deviceID string) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&model.DiscoveredDevice{}).Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":            model.StatusImported,
			"imported_at":       &now,
			"matched_device_id": &deviceID,
		}).Error
}
