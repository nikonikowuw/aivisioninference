// Package model 定义系统数据模型，包含 GORM 结构体和数据库表映射。
package model

import (
	"encoding/json"
	"time"

	"gorm.io/datatypes"
)

// 授权类型常量
const (
	LicenseTypeTrial     = "trial"
	LicenseTypeFormal    = "formal"
	LicenseTypePermanent = "permanent"
)

// 授权状态常量
const (
	LicenseStatusActive      = "active"
	LicenseStatusExpired     = "expired"
	LicenseStatusRevoked     = "revoked"
	LicenseStatusNotYetValid = "not_yet_valid"
)

// 授权过期动作常量
const (
	ExpireActionTerminate    = "terminate"    // 立即终止运行中任务
	ExpireActionKeepRunning  = "keep_running" // 允许运行中任务继续，仅拦截新任务
)

// DeviceLicense 表示设备授权/许可证，基于 JWT 离线验签并绑定设备指纹。
// 授权文件由外部系统签发，用户上传后在本地完成签名验证、指纹匹配和有效期校验。
type DeviceLicense struct {
	BaseModel
	LicenseID         string         `gorm:"type:varchar(128);uniqueIndex" json:"license_id"`                   // 授权 ID（来自 JWT payload）
	LicenseType       string         `gorm:"type:varchar(16);not null;default:formal" json:"license_type"`      // trial / formal / permanent
	DeviceSN          string         `gorm:"type:varchar(128);not null;index" json:"device_sn"`                 // 绑定设备序列号
	DeviceFingerprint string         `gorm:"type:varchar(128)" json:"device_fingerprint,omitempty"`             // 绑定设备指纹（SHA-256 Hash）
	Algorithms        datatypes.JSON `gorm:"type:jsonb;default:'[]'" json:"algorithms,omitempty"`               // 授权算法范围列表
	MaxStreams         int            `gorm:"default:0" json:"max_streams"`                                      // 全局最大授权视频流路数（0=不限）
	Features          datatypes.JSON `gorm:"type:jsonb;default:'[]'" json:"features,omitempty"`                 // 授权功能列表（如 face_identity, webhook）
	NotBefore         time.Time      `gorm:"not null;index" json:"not_before"`                                  // 授权生效时间
	NotAfter          *time.Time     `gorm:"index" json:"not_after,omitempty"`                                  // 授权到期时间；永久授权为空
	ExpireAction      string         `gorm:"type:varchar(16);not null;default:keep_running" json:"expire_action"` // 过期动作：terminate / keep_running
	Status            string         `gorm:"type:varchar(16);not null;default:active;index" json:"status"`       // active / expired / revoked / not_yet_valid
	Signature         string         `gorm:"type:text;not null" json:"-"`                                       // 原始 JWT 字符串（完整内容，用于审计和重新解析）
	RawContent        string         `gorm:"type:text" json:"-"`                                                // 原始上传文件内容
}

// IsExpired 判断授权是否已过期
func (dl *DeviceLicense) IsExpired() bool {
	if dl.NotAfter == nil {
		return false // 永久授权
	}
	return time.Now().After(*dl.NotAfter)
}

// IsEffective 判断授权当前是否生效（在有效期内）
func (dl *DeviceLicense) IsEffective() bool {
	return !time.Now().Before(dl.NotBefore) && !dl.IsExpired()
}

// HasAlgorithm 判断授权是否包含指定算法（支持通配符 "*"）
func (dl *DeviceLicense) HasAlgorithm(algoName string) bool {
	if len(dl.Algorithms) == 0 {
		return false
	}
	var algos []string
	if err := json.Unmarshal(dl.Algorithms, &algos); err != nil {
		return false
	}
	for _, a := range algos {
		if a == algoName || a == "*" {
			return true
		}
	}
	return false
}

// ShouldTerminateOnExpire 判断授权过期后是否应强制终止运行中任务
func (dl *DeviceLicense) ShouldTerminateOnExpire() bool {
	return dl.ExpireAction == ExpireActionTerminate
}

// SortableFields 返回允许排序的字段列表
func (DeviceLicense) SortableFields() []string {
	return []string{"created_at", "status", "not_before", "not_after"}
}

// TableName 覆盖 DeviceLicense 的默认表名
func (DeviceLicense) TableName() string {
	return "device_licenses"
}
