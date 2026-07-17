// Package dto 定义请求和响应的数据传输结构体，包含参数校验和序列化标签。
package dto

import "github.com/niko-admin/niko-admin/internal/pkg/scopes"

// DeviceListRequest 设备列表查询参数
type DeviceListRequest struct {
	PageRequest
	Keyword    string `form:"keyword"`
	Status     string `form:"status"`
	AccessType string `form:"access_type"`
	GroupID    string `form:"group_id"`
}

// FilterScopes 返回当前请求对应的 GORM 查询范围函数列表
func (r *DeviceListRequest) FilterScopes() []scopes.Scope {
	var sc []scopes.Scope
	if r.Keyword != "" {
		sc = append(sc, scopes.MultiLike([]string{"device_name", "manufacturer", "model"}, r.Keyword))
	}
	if r.Status != "" {
		sc = append(sc, scopes.Eq("status", r.Status))
	}
	if r.AccessType != "" {
		sc = append(sc, scopes.Eq("access_type", r.AccessType))
	}
	// GroupID 筛选在 Repository 层通过 JOIN 实现，此处不返回 Scope
	return sc
}

// DeviceCreateRequest 创建设备请求
type DeviceCreateRequest struct {
	DeviceName      string   `json:"device_name" binding:"required,min=1,max=128"`
	AccessType      string   `json:"access_type" binding:"required,oneof=rtsp gb28181 nvr_channel other"`
	RtspURL         string   `json:"rtsp_url" binding:"omitempty,max=1024"`
	GB28181DeviceID string   `json:"gb28181_device_id" binding:"omitempty,len=20"`
	GB28181ChannelID string  `json:"gb28181_channel_id" binding:"omitempty,max=64"`
	Username        string   `json:"username" binding:"max=128"`
	Password        string   `json:"password" binding:"omitempty,min=1,max=128"`
	Manufacturer    string   `json:"manufacturer" binding:"max=64"`
	Model           string   `json:"model" binding:"max=64"`
	FirmwareVersion string   `json:"firmware_version" binding:"max=64"`
	Latitude        *float64 `json:"latitude"`
	Longitude       *float64 `json:"longitude"`
	LocationDesc    string   `json:"location_desc" binding:"max=255"`
	Remark          string   `json:"remark" binding:"max=1000"`
	GroupIDs        []string `json:"group_ids"`
}

// DeviceUpdateRequest 更新设备请求
type DeviceUpdateRequest struct {
	DeviceName       string   `json:"device_name" binding:"omitempty,min=1,max=128"`
	AccessType       string   `json:"access_type" binding:"omitempty,oneof=rtsp gb28181 nvr_channel other"`
	RtspURL          string   `json:"rtsp_url" binding:"omitempty,max=1024"`
	GB28181DeviceID  string   `json:"gb28181_device_id" binding:"omitempty,len=20"`
	GB28181ChannelID string   `json:"gb28181_channel_id" binding:"omitempty,max=64"`
	Username         string   `json:"username" binding:"max=128"`
	Password         string   `json:"password" binding:"omitempty,min=1,max=128"`
	Manufacturer     string   `json:"manufacturer" binding:"max=64"`
	Model            string   `json:"model" binding:"max=64"`
	FirmwareVersion  string   `json:"firmware_version" binding:"max=64"`
	Latitude         *float64 `json:"latitude"`
	Longitude        *float64 `json:"longitude"`
	LocationDesc     string   `json:"location_desc" binding:"max=255"`
	Remark           string   `json:"remark" binding:"max=1000"`
	GroupIDs         []string `json:"group_ids"`
	Status           string   `json:"status" binding:"omitempty,oneof=unknown online offline error disabled"`
	Enabled          *bool    `json:"enabled"`
}

// DeviceResponse 设备响应（不包含 password 敏感字段）
type DeviceResponse struct {
	ID               string                `json:"id"`
	DeviceName       string                `json:"device_name"`
	AccessType       string                `json:"access_type"`
	RtspURL          string                `json:"rtsp_url,omitempty"`
	GB28181DeviceID  string                `json:"gb28181_device_id,omitempty"`
	GB28181ChannelID string                `json:"gb28181_channel_id,omitempty"`
	Username         string                `json:"username,omitempty"`
	Manufacturer     string                `json:"manufacturer,omitempty"`
	Model            string                `json:"model,omitempty"`
	FirmwareVersion  string                `json:"firmware_version,omitempty"`
	Status           string                `json:"status"`
	Enabled          bool                  `json:"enabled"`
	Latitude         *float64              `json:"latitude,omitempty"`
	Longitude        *float64              `json:"longitude,omitempty"`
	LocationDesc     string                `json:"location_desc,omitempty"`
	LastOnlineAt     *string               `json:"last_online_at,omitempty"`
	LastOfflineAt    *string               `json:"last_offline_at,omitempty"`
	LastErrorCode    string                `json:"last_error_code,omitempty"`
	LastErrorMessage string                `json:"last_error_message,omitempty"`
	ExternalKey      string                `json:"external_key,omitempty"`
	Remark           string                `json:"remark,omitempty"`
	Version          int                   `json:"version"`
	Groups           []DeviceGroupResponse `json:"groups,omitempty"`
	CreatedBy        *string               `json:"created_by,omitempty"`
	CreatedAt        string                `json:"created_at"`
	UpdatedAt        string                `json:"updated_at"`
}

// DeviceListResponse 设备列表响应项（精简版，不含敏感信息）
type DeviceListResponse struct {
	ID            string                `json:"id"`
	DeviceName    string                `json:"device_name"`
	AccessType    string                `json:"access_type"`
	Status        string                `json:"status"`
	Enabled       bool                  `json:"enabled"`
	LastOnlineAt  *string               `json:"last_online_at,omitempty"`
	LastOfflineAt *string               `json:"last_offline_at,omitempty"`
	Manufacturer  string                `json:"manufacturer,omitempty"`
	Groups        []DeviceGroupResponse `json:"groups,omitempty"`
	CreatedAt     string                `json:"created_at"`
	UpdatedAt     string                `json:"updated_at"`
}

// DeviceTestResultResponse 设备测试连接结果
type DeviceTestResultResponse struct {
	Success  bool   `json:"success"`
	Message  string `json:"message"`
	TestedAt string `json:"tested_at"`
}

// DeviceBatchImportResult 批量导入结果
type DeviceBatchImportResult struct {
	BatchResult
}

// DeviceGroupListRequest 设备分组列表查询参数
type DeviceGroupListRequest struct {
	PageRequest
	Keyword string `form:"keyword"`
}

// DeviceGroupCreateRequest 创建设备分组请求
type DeviceGroupCreateRequest struct {
	GroupName   string  `json:"group_name" binding:"required,min=1,max=128"`
	Description string  `json:"description" binding:"max=255"`
	ParentID    *string `json:"parent_id"`
	SortOrder   int     `json:"sort_order"`
}

// DeviceGroupUpdateRequest 更新设备分组请求
type DeviceGroupUpdateRequest struct {
	GroupName   string  `json:"group_name" binding:"omitempty,min=1,max=128"`
	Description string  `json:"description" binding:"max=255"`
	ParentID    *string `json:"parent_id"`
	SortOrder   int     `json:"sort_order"`
}

// DeviceGroupResponse 设备分组响应
type DeviceGroupResponse struct {
	ID          string `json:"id"`
	GroupName   string `json:"group_name"`
	Description string `json:"description,omitempty"`
	ParentID    string `json:"parent_id,omitempty"`
	SortOrder   int    `json:"sort_order"`
	DeviceCount int64  `json:"device_count"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}
