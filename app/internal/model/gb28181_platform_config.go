package model

import "time"

type GB28181PlatformConfig struct {
	ID               string    `gorm:"type:varchar(64);primaryKey" json:"id"` // Unique ID, usually "default"
	Enabled          bool      `gorm:"default:true" json:"enabled"`
	SipID            string    `gorm:"type:varchar(64);default:34020000002000000001" json:"sip_id"`
	SipDomain        string    `gorm:"type:varchar(128);default:3402000000" json:"sip_domain"`
	SipRealm         string    `gorm:"type:varchar(128);default:3402000000" json:"sip_realm"`
	SipPassword      string    `gorm:"type:text" json:"-"`
	ListenIP         string    `gorm:"type:varchar(64);default:0.0.0.0" json:"listen_ip"`
	ListenPort       int       `gorm:"default:5060" json:"listen_port"`
	Transport        string    `gorm:"type:varchar(32);default:udp" json:"transport"` // udp | tcp
	AdvertisedIP     string    `gorm:"type:varchar(255)" json:"advertised_ip,omitempty"`
	RtpIP            string    `gorm:"type:varchar(255)" json:"rtp_ip,omitempty"`
	HeartbeatTimeout int       `gorm:"default:180" json:"heartbeat_timeout"` // seconds
	CatalogInterval  int       `gorm:"default:3600" json:"catalog_interval"` // seconds
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func (GB28181PlatformConfig) TableName() string {
	return "gb28181_platform_configs"
}
