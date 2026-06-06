// Package dto 定义请求和响应的数据传输结构体，包含参数校验和序列化标签。
package dto

import (
	"time"

	"github.com/niko-admin/niko-admin/internal/pkg/scopes"
)

// LicenseUploadRequest 授权文件上传请求（使用 multipart/form-data）
type LicenseUploadRequest struct {
	// File 由 handler 层从 FormFile 获取，不在 DTO 中绑定
}

// LicenseInfoResponse 授权信息响应，返回给前端展示
type LicenseInfoResponse struct {
	ID               string     `json:"id"`
	LicenseID        string     `json:"license_id"`
	LicenseType      string     `json:"license_type"`
	DeviceSN         string     `json:"device_sn"`
	DeviceFingerprint string   `json:"device_fingerprint"`
	Algorithms       []string   `json:"algorithms,omitempty"`
	MaxStreams        int       `json:"max_streams"`
	Features         []string   `json:"features,omitempty"`
	NotBefore        time.Time  `json:"not_before"`
	NotAfter         *time.Time `json:"not_after,omitempty"`
	ExpireAction     string     `json:"expire_action"`
	Status           string     `json:"status"`
	RemainingDays    *int       `json:"remaining_days,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// LicenseListRequest 授权列表请求
type LicenseListRequest struct {
	PageRequest
	Keyword string `form:"keyword"`
	Status  string `form:"status"`
}

// FilterScopes 返回授权列表的过滤条件。
func (r *LicenseListRequest) FilterScopes() []scopes.Scope {
	var sc []scopes.Scope
	if r.Keyword != "" {
		sc = append(sc, scopes.MultiLike([]string{"license_id", "device_sn"}, r.Keyword))
	}
	if r.Status != "" {
		sc = append(sc, scopes.Eq("status", r.Status))
	}
	return sc
}

// FingerprintResponse 设备指纹查询响应
type FingerprintResponse struct {
	DeviceSN      string `json:"device_sn"`
	Fingerprint   string `json:"fingerprint"`
	HashAlgorithm string `json:"hash_algorithm"`
}
