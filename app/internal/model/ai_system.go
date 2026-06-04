// Package model 定义系统数据模型，包含 GORM 结构体和数据库表映射。
package model

import "time"

// AISystemConfig 表示 AIVisionInference 系统的 Key-Value 配置项
type AISystemConfig struct {
	ConfigKey   string    `gorm:"type:varchar(128);primaryKey" json:"config_key"`
	ConfigValue string    `gorm:"type:text;not null" json:"config_value"`
	ConfigType  string    `gorm:"type:varchar(16);not null;default:string;check:config_type IN ('string','int','float','bool','json')" json:"config_type"`
	Description string    `gorm:"type:varchar(255)" json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TableName 覆盖 AISystemConfig 的默认表名
func (AISystemConfig) TableName() string {
	return "ai_system_configs"
}
