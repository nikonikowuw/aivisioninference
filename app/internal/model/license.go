// Package model 定义系统数据模型，包含 GORM 结构体和数据库表映射。
package model

import "time"

// 设备授权常量定义
const (
	DeviceLicenseStatusActive  = "active"
	DeviceLicenseStatusExpired = "expired"
	DeviceLicenseStatusRevoked = "revoked"
)

// DeviceLicense 表示设备授权/许可证，支持离线验签和设备指纹绑定（MVP+ 可选）
type DeviceLicense struct {
	BaseModel
	DeviceSN     string     `gorm:"type:varchar(128);not null;index" json:"device_sn"`
	LicenseKey   string     `gorm:"type:text;not null" json:"-"`
	Algorithms   []string   `gorm:"type:jsonb;serializer:json" json:"algorithms,omitempty"`
	MaxConcurrent int       `gorm:"default:1" json:"max_concurrent"`
	MaxVersions  int        `gorm:"default:1" json:"max_versions"`
	IssuedAt     time.Time  `json:"issued_at"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
	IsPermanent  bool       `gorm:"default:false" json:"is_permanent"`
	Status       string     `gorm:"type:varchar(16);not null;default:active;index" json:"status"`
}

// SortableFields 返回允许排序的字段列表
func (DeviceLicense) SortableFields() []string {
	return []string{"created_at", "status", "issued_at"}
}

// TableName 覆盖 DeviceLicense 的默认表名
func (DeviceLicense) TableName() string {
	return "device_licenses"
}
