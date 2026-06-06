// Package model 定义系统数据模型，包含 GORM 结构体和数据库表映射。
package model

import (
	"time"

	"gorm.io/datatypes"
)

// 系统配置常量定义
const (
	// 语言配置
	LanguageZhCN = "zh-CN" // 简体中文
	LanguageZhTW = "zh-TW" // 繁体中文
	LanguageEn   = "en"    // 英语

	// 日志级别
	LogLevelDebug = "debug"
	LogLevelInfo  = "info"
	LogLevelWarn  = "warn"
	LogLevelError = "error"

	// 时间模式
	TimeModeNTP    = "ntp"
	TimeModeManual = "manual"
)

// SystemConfig 表示系统基础配置
type SystemConfig struct {
	ID             string    `gorm:"type:varchar(64);primaryKey" json:"id"` // 配置项 ID
	DeviceSN       string    `gorm:"type:varchar(64);uniqueIndex" json:"device_sn"`
	DeviceName     string    `gorm:"type:varchar(128)" json:"device_name"`
	DeployLocation string    `gorm:"type:varchar(255)" json:"deploy_location"`
	Language       string    `gorm:"type:varchar(16);default:zh-CN" json:"language"`
	LogLevel       string    `gorm:"type:varchar(16);default:info" json:"log_level"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// TableName 覆盖 SystemConfig 的默认表名
func (SystemConfig) TableName() string {
	return "system_configs"
}

// NetworkInterfaceConfig 表示网络接口持久化配置
type NetworkInterfaceConfig struct {
	BaseModel
	InterfaceName string         `gorm:"type:varchar(32);uniqueIndex;not null" json:"interface_name"`
	ConfigMode    string         `gorm:"type:varchar(16);not null;default:dhcp;check:config_mode IN ('dhcp','static')" json:"config_mode"`
	IPAddress     string         `gorm:"type:varchar(45)" json:"ip_address,omitempty"`
	CIDRPrefix    int            `gorm:"type:smallint" json:"cidr_prefix,omitempty"`
	Gateway       string         `gorm:"type:varchar(45)" json:"gateway,omitempty"`
	DNSServers    datatypes.JSON `gorm:"type:jsonb;default:'[]'" json:"dns_servers,omitempty"`
	Enabled       bool           `gorm:"default:true" json:"enabled"`
	Description   string         `gorm:"type:varchar(255)" json:"description,omitempty"`
}

// SortableFields 返回允许排序的字段列表
func (NetworkInterfaceConfig) SortableFields() []string {
	return []string{"created_at", "interface_name"}
}

// TableName 覆盖 NetworkInterfaceConfig 的默认表名
func (NetworkInterfaceConfig) TableName() string {
	return "network_interface_configs"
}

// TimeConfig 表示时间配置
type TimeConfig struct {
	ID          string         `gorm:"type:varchar(64);primaryKey" json:"id"`
	TimeMode    string         `gorm:"type:varchar(16);not null;default:ntp;check:time_mode IN ('ntp','manual')" json:"time_mode"`
	Timezone    string         `gorm:"type:varchar(64);not null;default:Asia/Shanghai" json:"timezone"`
	NTPEnabled  bool           `gorm:"default:true" json:"ntp_enabled"`
	NTPServers  datatypes.JSON `gorm:"type:jsonb;default:'[\"ntp.aliyun.com\",\"cn.ntp.org.cn\"]'" json:"ntp_servers"`
	SyncInterval int           `gorm:"default:60" json:"sync_interval"` // 同步间隔(分钟)
	LastSyncAt  *time.Time     `json:"last_sync_at,omitempty"`
	LastSyncStatus string      `gorm:"type:varchar(16)" json:"last_sync_status,omitempty"` // success/failed
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

// TableName 覆盖 TimeConfig 的默认表名
func (TimeConfig) TableName() string {
	return "time_configs"
}

// NTPServerStatus 表示 NTP 服务器状态（非持久化，运行时数据）
type NTPServerStatus struct {
	Host      string     `json:"host"`
	Port      int        `json:"port"`
	Status    string     `json:"status"` // reachable/unreachable
	Latency   int64      `json:"latency_ms"`
	LastCheck *time.Time `json:"last_check,omitempty"`
}
