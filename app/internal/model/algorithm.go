// Package model 定义系统数据模型，包含 GORM 结构体和数据库表映射。
package model

import (
	"time"

	"github.com/lib/pq"
	"gorm.io/datatypes"
)

// 算法包常量定义
const (
	SelfCheckStatusPending = "pending"
	SelfCheckStatusRunning = "running"
	SelfCheckStatusPassed  = "passed"
	SelfCheckStatusFailed  = "failed"

	AlgoPackageStatusDraft    = "draft"
	AlgoPackageStatusEnabled  = "enabled"
	AlgoPackageStatusDisabled = "disabled"
	AlgoPackageStatusArchived = "archived"
)

// CategoryCode 表示平台统一五位数字类别编码（10000-99999），提供 i18n 展示名
type CategoryCode struct {
	CategoryCode    int       `gorm:"primaryKey;check:category_code >= 10000 AND category_code <= 99999" json:"category_code"`
	DisplayName     string    `gorm:"type:varchar(128);not null" json:"display_name"`
	DisplayNameEN   string    `gorm:"type:varchar(128)" json:"display_name_en,omitempty"`
	DisplayNameZhTW string    `gorm:"type:varchar(128)" json:"display_name_zh_tw,omitempty"`
	Domain          string    `gorm:"type:varchar(64);index" json:"domain,omitempty"`
	Description     string    `gorm:"type:varchar(255)" json:"description,omitempty"`
	IsSystem        bool      `gorm:"default:false" json:"is_system"`
	Enabled         bool      `gorm:"default:true" json:"enabled"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// SortableFields 返回允许排序的字段列表
func (CategoryCode) SortableFields() []string {
	return []string{"category_code", "domain", "display_name"}
}

// TableName 覆盖 CategoryCode 的默认表名
func (CategoryCode) TableName() string {
	return "category_codes"
}

// AlgorithmPackage 表示算法包，支持版本管理、自检、引用计数和热更新
type AlgorithmPackage struct {
	BaseModel
	AlgorithmName     string         `gorm:"type:varchar(128);not null;index" json:"algorithm_name"`
	AlgorithmAlias    string         `gorm:"type:varchar(128)" json:"algorithm_alias,omitempty"`
	Version           string         `gorm:"type:varchar(32);not null" json:"version"`
	Domain            string         `gorm:"type:varchar(64);not null;index" json:"domain"`
	ResultSchema      string         `gorm:"type:varchar(64);not null" json:"result_schema"`
	CapabilitiesImage pq.StringArray `gorm:"type:text[]" json:"capabilities_image,omitempty"`
	CapabilitiesData  pq.StringArray `gorm:"type:text[]" json:"capabilities_data,omitempty"`
	Hardware          pq.StringArray `gorm:"type:text[]" json:"hardware,omitempty"`
	Description       string         `gorm:"type:text" json:"description,omitempty"`
	PackagePath       string         `gorm:"type:varchar(512);not null" json:"package_path"`
	ExtractPath       string         `gorm:"type:varchar(512);not null" json:"extract_path"`
	PackageSize       int64          `json:"package_size,omitempty"`
	PackageMD5        string         `gorm:"type:varchar(64)" json:"package_md5,omitempty"`
	SoPath            string         `gorm:"type:varchar(512);not null" json:"so_path"`
	AIParamsSchema    datatypes.JSON `gorm:"type:jsonb" json:"ai_params_schema,omitempty"`
	SelfCheckStatus   string         `gorm:"type:varchar(16);not null;default:pending" json:"self_check_status"`
	SelfCheckResult   datatypes.JSON `gorm:"type:jsonb" json:"self_check_result,omitempty"`
	SelfCheckAt       *time.Time     `json:"self_check_at,omitempty"`
	Status            string         `gorm:"type:varchar(16);not null;default:draft;index" json:"status"`
	RefCount          int            `gorm:"default:0" json:"ref_count"`
	IsCurrent         bool           `gorm:"default:false;index" json:"is_current"`
	Remark            string         `gorm:"type:text" json:"remark,omitempty"`
}

// SortableFields 返回允许排序的字段列表
func (AlgorithmPackage) SortableFields() []string {
	return []string{"created_at", "version", "algorithm_name", "status"}
}

// AlgorithmLabelMap 表示算法包类别映射，将模型内部 label_id 映射为系统 category_code
type AlgorithmLabelMap struct {
	BaseModel
	PackageID    string `gorm:"type:uuid;uniqueIndex:idx_algo_label_package_label;not null" json:"package_id"`
	LabelID      int    `gorm:"uniqueIndex:idx_algo_label_package_label;not null" json:"label_id"`
	CategoryCode int    `gorm:"index;not null" json:"category_code"`
	DisplayName  string `gorm:"type:varchar(128)" json:"display_name,omitempty"`
}

// SortableFields 返回允许排序的字段列表
func (AlgorithmLabelMap) SortableFields() []string {
	return []string{"created_at", "label_id"}
}
