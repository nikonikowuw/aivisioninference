// Package model 定义系统数据模型，包含 GORM 结构体和数据库表映射。
package model

// BrandConfig 存储系统级品牌展示配置。
type BrandConfig struct {
	BaseModel
	SystemName string `gorm:"type:varchar(128);not null;default:'Niko Admin'" json:"system_name"` // 系统显示名称
	LogoURL    string `gorm:"type:varchar(512);not null;default:'/favicon.ico'" json:"logo_url"`  // 系统 Logo URL
}

// SortableFields 返回允许排序的字段列表。
func (BrandConfig) SortableFields() []string {
	return []string{"created_at", "updated_at"}
}
