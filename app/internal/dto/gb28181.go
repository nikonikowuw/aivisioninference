package dto

import "time"

// GB28181DeviceListRequest GB28181 设备列表请求
type GB28181DeviceListRequest struct {
	PageRequest
	Keyword string `form:"keyword"` // 搜索关键词 (DeviceCode, Manufacturer, Model等)
	Status  string `form:"status"`  // 状态筛选
}

// GB28181DeviceResponse GB28181 设备响应
type GB28181DeviceResponse struct {
	ID                string     `json:"id"`
	DeviceCode        string     `json:"device_code"`
	RegisterAddress   string     `json:"register_address"`
	RegisterPort      int        `json:"register_port"`
	SipID             string     `json:"sip_id"`
	SipDomain         string     `json:"sip_domain"`
	LastRegisterAt    *time.Time `json:"last_register_at"`
	LastHeartbeatAt   *time.Time `json:"last_heartbeat_at"`
	LastCatalogAt     *time.Time `json:"last_catalog_at"`
	HeartbeatInterval int        `json:"heartbeat_interval"`
	Status            string     `json:"status"`
	ChannelCount      int        `json:"channel_count"`
	Manufacturer      string     `json:"manufacturer"`
	Model             string     `json:"model"`
	Firmware          string     `json:"firmware"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

// GB28181DeviceCreateRequest 创建请求
type GB28181DeviceCreateRequest struct {
	DeviceCode        string `json:"device_code" validate:"required,len=20"`
	SipID             string `json:"sip_id" validate:"omitempty"`
	SipDomain         string `json:"sip_domain" validate:"required"`
	SipPassword       string `json:"sip_password" validate:"omitempty"`
	HeartbeatInterval int    `json:"heartbeat_interval" validate:"omitempty,min=10,max=300"`
}

// GB28181DeviceUpdateRequest 更新请求
type GB28181DeviceUpdateRequest struct {
	SipID             *string `json:"sip_id" validate:"omitempty"`
	SipDomain         *string `json:"sip_domain" validate:"omitempty"`
	SipPassword       *string `json:"sip_password" validate:"omitempty"` // 留空表示不改
	HeartbeatInterval *int    `json:"heartbeat_interval" validate:"omitempty,min=10,max=300"`
}

// CatalogTaskResponse 目录查询任务响应
type CatalogTaskResponse struct {
	TaskID string `json:"task_id"`
}

// CatalogTaskStatusResponse 目录查询任务状态响应
type CatalogTaskStatusResponse struct {
	TaskID       string `json:"task_id"`
	Status       string `json:"status"` // pending, completed, failed
	ChannelCount int    `json:"channel_count"`
	Error        string `json:"error,omitempty"`
}

// GB28181PlatformConfigResponse GB28181 平台配置响应
type GB28181PlatformConfigResponse struct {
	Enabled          bool      `json:"enabled"`
	SipID            string    `json:"sip_id"`
	SipDomain        string    `json:"sip_domain"`
	SipRealm         string    `json:"sip_realm"`
	ListenIP         string    `json:"listen_ip"`
	ListenPort       int       `json:"listen_port"`
	Transport        string    `json:"transport"` // udp | tcp
	AdvertisedIP     string    `json:"advertised_ip"`
	RtpIP            string    `json:"rtp_ip"`
	HeartbeatTimeout int       `json:"heartbeat_timeout"`
	CatalogInterval  int       `json:"catalog_interval"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// GB28181PlatformConfigRequest GB28181 平台配置请求
type GB28181PlatformConfigRequest struct {
	Enabled          bool   `json:"enabled"`
	SipID            string `json:"sip_id" validate:"required,len=20"`
	SipDomain        string `json:"sip_domain" validate:"required"`
	SipRealm         string `json:"sip_realm" validate:"required"`
	SipPassword      string `json:"sip_password" validate:"omitempty"` // Empty means keep unchanged
	ListenIP         string `json:"listen_ip" validate:"required"`
	ListenPort       int    `json:"listen_port" validate:"required,min=1024,max=65535"`
	Transport        string `json:"transport" validate:"required,oneof=udp tcp"`
	AdvertisedIP     string `json:"advertised_ip"`
	RtpIP            string `json:"rtp_ip"`
	HeartbeatTimeout int    `json:"heartbeat_timeout" validate:"required,min=10,max=1800"`
	CatalogInterval  int    `json:"catalog_interval" validate:"required,min=10,max=86400"`
}
