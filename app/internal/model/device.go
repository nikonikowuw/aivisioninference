// Package model 定义系统数据模型，包含 GORM 结构体和数据库表映射。
package model

import (
	"time"
)

// 设备常量定义
const (
	DeviceAccessTypeRTSP       = "rtsp"
	DeviceAccessTypeGB28181    = "gb28181"
	DeviceAccessTypeGB28181NVR = "gb28181_nvr"
	DeviceAccessTypeNVRChannel = "nvr_channel"
	DeviceAccessTypeOther      = "other"

	DeviceStatusUnknown  = "unknown"
	DeviceStatusOnline   = "online"
	DeviceStatusOffline  = "offline"
	DeviceStatusError    = "error"
	DeviceStatusDisabled = "disabled"

	GB28181StatusRegistered = "registered"
	GB28181StatusOnline     = "online"
	GB28181StatusOffline    = "offline"
)

// DeviceGroup 表示设备分组，支持单层分组（一个设备可属于多个分组）
type DeviceGroup struct {
	BaseModel
	GroupName   string        `gorm:"type:varchar(128);not null" json:"group_name"`
	Description string        `gorm:"type:varchar(255)" json:"description"`
	ParentID    *string       `gorm:"type:uuid" json:"parent_id"`
	Parent      *DeviceGroup  `gorm:"foreignKey:ParentID" json:"parent,omitempty"`
	Children    []DeviceGroup `gorm:"foreignKey:ParentID" json:"children,omitempty"`
	SortOrder   int           `gorm:"default:0" json:"sort_order"`
}

// SortableFields 返回允许排序的字段列表
func (DeviceGroup) SortableFields() []string {
	return []string{"created_at", "sort_order", "group_name"}
}

// Device 表示流设备，支持 RTSP、GB28181 等多种接入方式
// 一机一任务约束：每个设备只能有一个激活的推理任务
type Device struct {
	BaseModel
	DeviceName       string        `gorm:"type:varchar(128);not null;index" json:"device_name"`
	AccessType       string        `gorm:"type:varchar(16);not null;check:access_type IN ('rtsp','gb28181','nvr_channel','other')" json:"access_type"`
	RtspURL          string        `gorm:"type:text" json:"rtsp_url,omitempty"`
	GB28181DeviceID  string        `gorm:"type:varchar(64);index" json:"gb28181_device_id,omitempty"`
	GB28181ChannelID string        `gorm:"type:varchar(64)" json:"gb28181_channel_id,omitempty"`
	Username         string        `gorm:"type:varchar(128)" json:"username,omitempty"`
	Password         string        `gorm:"type:text" json:"-"`
	Manufacturer     string        `gorm:"type:varchar(64)" json:"manufacturer,omitempty"`
	Model            string        `gorm:"type:varchar(64)" json:"model,omitempty"`
	FirmwareVersion  string        `gorm:"type:varchar(64)" json:"firmware_version,omitempty"`
	Status           string        `gorm:"type:varchar(16);not null;default:unknown;index" json:"status"`
	Enabled          bool          `gorm:"default:true;index" json:"enabled"`
	Latitude         *float64      `gorm:"type:numeric(10,7)" json:"latitude,omitempty"`
	Longitude        *float64      `gorm:"type:numeric(10,7)" json:"longitude,omitempty"`
	LocationDesc     string        `gorm:"type:varchar(255)" json:"location_desc,omitempty"`
	LastOnlineAt     *time.Time    `json:"last_online_at,omitempty"`
	LastOfflineAt    *time.Time    `json:"last_offline_at,omitempty"`
	LastErrorCode    string        `gorm:"type:varchar(64)" json:"last_error_code,omitempty"`
	LastErrorMessage string        `gorm:"type:text" json:"last_error_message,omitempty"`
	ExternalKey      *string       `gorm:"type:varchar(128);uniqueIndex" json:"external_key,omitempty"`
	Remark           string        `gorm:"type:text" json:"remark,omitempty"`
	Version          int           `gorm:"default:1" json:"version"`
	AutoInfer        bool          `gorm:"default:false" json:"auto_infer"`
	ParentNvrID      *string       `gorm:"type:uuid;index" json:"parent_nvr_id,omitempty"`
	Groups           []DeviceGroup `gorm:"many2many:device_group_members;" json:"groups,omitempty"`
}

// SortableFields 返回允许排序的字段列表
func (Device) SortableFields() []string {
	return []string{"created_at", "updated_at", "device_name", "status"}
}

// TableName 覆盖 Device 的默认表名
func (Device) TableName() string {
	return "devices"
}

// DeviceGroupMember 是设备与分组多对多关系的关联表
type DeviceGroupMember struct {
	DeviceID  string    `gorm:"type:uuid;primaryKey" json:"device_id"`
	GroupID   string    `gorm:"type:uuid;primaryKey" json:"group_id"`
	CreatedAt time.Time `json:"created_at"`
}

// GB28181Device 表示 GB28181 国标设备注册、心跳和目录信息
type GB28181Device struct {
	BaseModel
	DeviceID          *string    `gorm:"type:uuid;index" json:"device_id,omitempty"`
	DeviceCode        string     `gorm:"type:varchar(64);uniqueIndex;not null" json:"device_code"`
	RegisterAddress   string     `gorm:"type:varchar(255)" json:"register_address,omitempty"`
	RegisterPort      int        `json:"register_port,omitempty"`
	SipID             string     `gorm:"type:varchar(64)" json:"sip_id,omitempty"`
	SipDomain         string     `gorm:"type:varchar(128)" json:"sip_domain,omitempty"`
	SipPassword       string     `gorm:"type:text" json:"-"`
	LastRegisterAt    *time.Time `json:"last_register_at,omitempty"`
	LastHeartbeatAt   *time.Time `json:"last_heartbeat_at,omitempty" gorm:"index"`
	LastCatalogAt     *time.Time `json:"last_catalog_at,omitempty"`
	HeartbeatInterval int        `gorm:"default:60" json:"heartbeat_interval"`
	Status            string     `gorm:"type:varchar(16);default:offline;index" json:"status"`
	ChannelCount      int        `gorm:"default:0" json:"channel_count"`
	Manufacturer      string     `gorm:"type:varchar(128)" json:"manufacturer,omitempty"`
	Model             string     `gorm:"type:varchar(64)" json:"model,omitempty"`
	Firmware          string     `gorm:"type:varchar(64)" json:"firmware,omitempty"`
	ExternalKey       string     `gorm:"type:varchar(200);uniqueIndex" json:"external_key,omitempty"`
}

// SortableFields 返回允许排序的字段列表
func (GB28181Device) SortableFields() []string {
	return []string{"created_at", "device_code", "status"}
}
