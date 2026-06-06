// Package model 定义系统数据模型，包含 GORM 结构体和数据库表映射。
package model

import (
	"time"

	"gorm.io/datatypes"
)

// 设备发现来源常量
const (
	SourceONVIF   = "onvif"
	SourceGB28181 = "gb28181"
	SourceNVR     = "nvr"
	SourceScan    = "scan"
)

// 设备发现状态常量
const (
	StatusPending  = "pending"
	StatusImported = "imported"
	StatusIgnored  = "ignored"
	StatusExpired  = "expired"
)

// DiscoveredDevice 表示设备发现结果暂存区记录
type DiscoveredDevice struct {
	BaseModel
	Source          string         `gorm:"type:varchar(16);not null;index" json:"source"`
	DeviceName      string         `gorm:"type:varchar(128)" json:"device_name"`
	DeviceIP        string         `gorm:"type:varchar(45);index" json:"device_ip"` // IPv4 or IPv6
	DeviceMAC       string         `gorm:"type:varchar(17);index" json:"device_mac"`
	Manufacturer    string         `gorm:"type:varchar(64)" json:"manufacturer"`
	Model           string         `gorm:"type:varchar(64)" json:"model"`
	FirmwareVersion string         `gorm:"type:varchar(64)" json:"firmware_version"`
	AccessType      string         `gorm:"type:varchar(16)" json:"access_type"` // rtsp/gb28181/nvr_channel
	AccessURL       string         `gorm:"type:text" json:"access_url"`          // 推定 RTSP URL
	GB28181Code     string         `gorm:"type:varchar(64);index" json:"gb28181_code"`
	NvrDeviceID     *string        `gorm:"type:uuid;index" json:"nvr_device_id,omitempty"`
	ExtraInfo       datatypes.JSON `gorm:"type:jsonb" json:"extra_info,omitempty"`
	Status          string         `gorm:"type:varchar(16);default:pending;index" json:"status"`
	ImportedAt      *time.Time     `json:"imported_at,omitempty"`
	IgnoredAt       *time.Time     `json:"ignored_at,omitempty"`
	MatchedDeviceID *string        `gorm:"type:uuid;index" json:"matched_device_id,omitempty"`
}

// TableName 覆盖 DiscoveredDevice 的默认表名
func (DiscoveredDevice) TableName() string {
	return "discovered_devices"
}

// SortableFields 返回允许排序的字段列表
func (DiscoveredDevice) SortableFields() []string {
	return []string{"created_at", "discovered_at", "status", "source"}
}
