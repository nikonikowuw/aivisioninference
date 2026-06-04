// Package repository 提供数据访问层实现，封装 GORM 数据库操作。
package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/model"
	"github.com/niko-admin/niko-admin/internal/pkg/scopes"
)

// DeviceRepository 处理 Device 流设备模型的数据持久化操作
type DeviceRepository struct {
	db *gorm.DB
}

// NewDeviceRepository 创建并返回一个新的 DeviceRepository 实例
func NewDeviceRepository(db *gorm.DB) *DeviceRepository {
	return &DeviceRepository{db: db}
}

// WithTx 返回一个绑定了指定事务的 DeviceRepository 实例
func (r *DeviceRepository) WithTx(tx *gorm.DB) *DeviceRepository {
	return &DeviceRepository{db: tx}
}

// Transaction 在数据库事务中执行 fn
func (r *DeviceRepository) Transaction(ctx context.Context, fn func(txRepo *DeviceRepository) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(r.WithTx(tx))
	})
}

// List 返回分页的设备列表，支持关键字、状态、接入类型筛选和排序
func (r *DeviceRepository) List(ctx context.Context, req dto.DeviceListRequest) ([]model.Device, int64, error) {
	var items []model.Device
	var total int64

	db := r.db.WithContext(ctx).Model(&model.Device{})

	// 应用筛选条件
	sc := req.FilterScopes()
	for _, s := range sc {
		db = s(db)
	}

	// 按分组筛选
	if req.GroupID != "" {
		db = db.Joins("JOIN device_group_members ON device_group_members.device_id = devices.id").
			Where("device_group_members.group_id = ?", req.GroupID)
	}

	// 统计总数
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if total == 0 {
		return items, 0, nil
	}

	// 排序和分页
	orderSc := scopes.OrderBy(req.Sort, req.Order, "created_at", "updated_at", "device_name", "status")
	orderDefault := scopes.OrderByDefault()
	paginate := scopes.Paginate(req.GetPage(), req.GetPageSize())

	// 预加载分组信息
	if err := db.Scopes(orderSc, orderDefault, paginate).Preload("Groups").Find(&items).Error; err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

// FindByID 根据设备 ID 查询设备
func (r *DeviceRepository) FindByID(ctx context.Context, id string) (*model.Device, error) {
	var item model.Device
	err := r.db.WithContext(ctx).Preload("Groups").Where("id = ?", id).First(&item).Error
	return &item, err
}

// FindByIDs 批量查询设备
func (r *DeviceRepository) FindByIDs(ctx context.Context, ids []string) ([]model.Device, error) {
	var items []model.Device
	err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&items).Error
	return items, err
}

// ExistsByName 检查设备名称是否已存在（排除指定 ID）
func (r *DeviceRepository) ExistsByName(ctx context.Context, name, excludeID string) (bool, error) {
	var count int64
	db := r.db.WithContext(ctx).Model(&model.Device{}).Where("device_name = ?", name)
	if excludeID != "" {
		db = db.Where("id != ?", excludeID)
	}
	err := db.Count(&count).Error
	return count > 0, err
}

// Create 插入一条新的设备记录
func (r *DeviceRepository) Create(ctx context.Context, item *model.Device) error {
	return r.db.WithContext(ctx).Create(item).Error
}

// Update 保存设备记录的所有修改
func (r *DeviceRepository) Update(ctx context.Context, item *model.Device) error {
	return r.db.WithContext(ctx).Save(item).Error
}

// UpdateStatus 更新设备状态和最近错误信息
func (r *DeviceRepository) UpdateStatus(ctx context.Context, id, status, errorCode, errorMessage string) error {
	updates := map[string]interface{}{
		"status": status,
	}
	if errorCode != "" {
		updates["last_error_code"] = errorCode
	}
	if errorMessage != "" {
		updates["last_error_message"] = errorMessage
	}
	return r.db.WithContext(ctx).Model(&model.Device{}).Where("id = ?", id).Updates(updates).Error
}

// Delete 软删除设备
func (r *DeviceRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&model.Device{}, "id = ?", id).Error
}

// BatchDelete 批量软删除设备
func (r *DeviceRepository) BatchDelete(ctx context.Context, ids []string) error {
	return r.db.WithContext(ctx).Delete(&model.Device{}, "id IN ?", ids).Error
}

// ListByGroupID 按分组查询设备列表（不含分页，用于分组详情展示）
func (r *DeviceRepository) ListByGroupID(ctx context.Context, groupID string) ([]model.Device, error) {
	var items []model.Device
	err := r.db.WithContext(ctx).
		Joins("JOIN device_group_members ON device_group_members.device_id = devices.id").
		Where("device_group_members.group_id = ?", groupID).
		Preload("Groups").
		Find(&items).Error
	return items, err
}

// DeviceGroupRepository 处理 DeviceGroup 设备分组模型的数据持久化操作
type DeviceGroupRepository struct {
	db *gorm.DB
}

// NewDeviceGroupRepository 创建并返回一个新的 DeviceGroupRepository 实例
func NewDeviceGroupRepository(db *gorm.DB) *DeviceGroupRepository {
	return &DeviceGroupRepository{db: db}
}

// List 返回分页的设备分组列表，包含每个分组的设备数量
func (r *DeviceGroupRepository) List(ctx context.Context, req dto.DeviceGroupListRequest) ([]model.DeviceGroup, int64, error) {
	var items []model.DeviceGroup
	var total int64

	db := r.db.WithContext(ctx).Model(&model.DeviceGroup{})

	if req.Keyword != "" {
		db = db.Where("group_name ILIKE ?", "%"+req.Keyword+"%")
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if total == 0 {
		return items, 0, nil
	}

	orderSc := scopes.OrderBy(req.Sort, req.Order, "created_at", "sort_order", "group_name")
	orderDefault := scopes.OrderByDefault()
	paginate := scopes.Paginate(req.GetPage(), req.GetPageSize())

	if err := db.Scopes(orderSc, orderDefault, paginate).Find(&items).Error; err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

// FindByID 根据分组 ID 查询设备分组
func (r *DeviceGroupRepository) FindByID(ctx context.Context, id string) (*model.DeviceGroup, error) {
	var item model.DeviceGroup
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&item).Error
	return &item, err
}

// Create 插入一条新的设备分组记录
func (r *DeviceGroupRepository) Create(ctx context.Context, item *model.DeviceGroup) error {
	return r.db.WithContext(ctx).Create(item).Error
}

// Update 保存设备分组记录的所有修改
func (r *DeviceGroupRepository) Update(ctx context.Context, item *model.DeviceGroup) error {
	return r.db.WithContext(ctx).Save(item).Error
}

// Delete 删除设备分组
func (r *DeviceGroupRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&model.DeviceGroup{}, "id = ?", id).Error
}

// CountByGroupID 获取指定分组下的设备数量
func (r *DeviceGroupRepository) CountByGroupID(ctx context.Context, groupID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.DeviceGroupMember{}).
		Where("group_id = ?", groupID).
		Count(&count).Error
	return count, err
}
