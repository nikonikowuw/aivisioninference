package model

import "time"

// DeviceSipConfig 存储 GB28181 设备的 SIP 通信参数。
// 一条记录对应一个 GB28181 NVR/平台设备（Device.AccessType=gb28181_nvr）。
type DeviceSipConfig struct {
	ID                string     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	DeviceID          string     `gorm:"type:uuid;uniqueIndex;not null" json:"device_id"`
	DeviceCode        string     `gorm:"type:varchar(64);uniqueIndex;not null" json:"device_code"`
	SipID             string     `gorm:"type:varchar(64)" json:"sip_id,omitempty"`
	SipDomain         string     `gorm:"type:varchar(128)" json:"sip_domain,omitempty"`
	SipPassword       string     `gorm:"type:text" json:"-"`
	RegisterAddress   string     `gorm:"type:varchar(255)" json:"register_address,omitempty"`
	RegisterPort      int        `json:"register_port,omitempty"`
	LastRegisterAt    *time.Time `json:"last_register_at,omitempty"`
	LastHeartbeatAt   *time.Time `json:"last_heartbeat_at,omitempty" gorm:"index"`
	LastCatalogAt     *time.Time `json:"last_catalog_at,omitempty"`
	HeartbeatInterval int        `gorm:"default:60" json:"heartbeat_interval"`
	ChannelCount      int        `gorm:"default:0" json:"channel_count"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

func (DeviceSipConfig) TableName() string {
	return "device_sip_configs"
}
