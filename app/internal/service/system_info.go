// Package service 提供系统信息管理（设备型号、部署位置等可由管理员配置的元数据）
package service

import (
	"errors"

	"github.com/niko-admin/niko-admin/internal/model"
	"gorm.io/gorm"
)

// SystemInfoService 系统信息管理服务
type SystemInfoService struct {
	db *gorm.DB
}

// NewSystemInfoService 创建系统信息服务
func NewSystemInfoService(db *gorm.DB) *SystemInfoService {
	return &SystemInfoService{db: db}
}

// SystemInfo 管理员可配置的系统信息
type SystemInfo struct {
	DeviceModel    string `json:"device_model"`
	DeployLocation string `json:"deploy_location"`
	Description    string `json:"description,omitempty"`
}

// GetSystemInfo 读取管理员配置的系统信息
func (s *SystemInfoService) GetSystemInfo() (*SystemInfo, error) {
	info := &SystemInfo{}

	if s.db == nil {
		return info, nil
	}

	var entries []model.AISystemConfig
	if err := s.db.Where("config_key IN ?", []string{modelKeyDeviceModel, "deploy_location", "system_description"}).Find(&entries).Error; err != nil {
		return nil, err
	}
	for _, e := range entries {
		switch e.ConfigKey {
		case modelKeyDeviceModel:
			info.DeviceModel = e.ConfigValue
		case "deploy_location":
			info.DeployLocation = e.ConfigValue
		case "system_description":
			info.Description = e.ConfigValue
		}
	}
	return info, nil
}

// UpdateSystemInfo 更新系统信息（upsert 行为）
func (s *SystemInfoService) UpdateSystemInfo(info *SystemInfo) error {
	if s.db == nil {
		return errors.New("db not initialized")
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := upsertKV(tx, modelKeyDeviceModel, info.DeviceModel, "string", "设备型号（管理员可手动覆盖硬件自动检测）"); err != nil {
			return err
		}
		if err := upsertKV(tx, "deploy_location", info.DeployLocation, "string", "部署位置描述"); err != nil {
			return err
		}
		if err := upsertKV(tx, "system_description", info.Description, "string", "系统描述"); err != nil {
			return err
		}
		return nil
	})
}

// upsertKV 插入或更新 K-V 配置
func upsertKV(tx *gorm.DB, key, value, ctype, desc string) error {
	var existing model.AISystemConfig
	err := tx.Where("config_key = ?", key).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// 不存在则创建（允许空值，以便清除之前的设置）
		return tx.Create(&model.AISystemConfig{
			ConfigKey:   key,
			ConfigValue: value,
			ConfigType:  ctype,
			Description: desc,
		}).Error
	}
	if err != nil {
		return err
	}
	existing.ConfigValue = value
	existing.ConfigType = ctype
	existing.Description = desc
	return tx.Save(&existing).Error
}

// SetDeviceModel 单独设置设备型号（便捷方法）
func (s *SystemInfoService) SetDeviceModel(model string) error {
	if s.db == nil {
		return errors.New("db not initialized")
	}
	return upsertKV(s.db, modelKeyDeviceModel, model, "string", "设备型号（管理员可手动覆盖硬件自动检测）")
}
