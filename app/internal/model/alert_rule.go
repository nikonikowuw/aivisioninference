package model

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// AlertRule 告警规则模型
// 定义何时触发告警以及如何通知
type AlertRule struct {
	BaseModel
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Name        string `gorm:"type:varchar(255);not null;comment:规则名称" json:"name"`
	NodeID      *string `gorm:"type:uuid;index;comment:关联边缘节点ID, NULL表示全局规则" json:"node_id,omitempty"`
	MetricType  string `gorm:"type:varchar(50);not null;comment:指标类型(cpu_usage/memory_usage/disk_usage/node_offline/node_error/temperature)" json:"metric_type"`
	Operator    string `gorm:"type:varchar(5);not null;comment:比较操作符(>/>=/</<=/==)" json:"operator"`
	Threshold   float64 `gorm:"not null;comment:阈值" json:"threshold"`
	DurationSeconds int `gorm:"not null;default:0;comment:持续时长(秒),条件需维持该时长才触发" json:"duration_seconds"`
	SilenceMinutes int `gorm:"not null;default:60;comment:静默期(分钟),触发后N分钟内不重复通知" json:"silence_minutes"`
	Enabled     bool    `gorm:"not null;default:true;comment:是否启用" json:"enabled"`
	NotifyChannels datatypes.JSON `gorm:"type:jsonb;not null;default:'[]';comment:通知渠道列表(webhook/email/telegram/dingtalk/feishu/wecom)" json:"notify_channels"`
	Description string  `gorm:"type:text;comment:规则描述" json:"description"`

	UpdatedAt time.Time `gorm:"comment:更新时间" json:"updated_at"`
}

// TableName 指定表名
func (AlertRule) TableName() string {
	return "alert_rules"
}

// SortableFields 返回允许排序的字段列表
func (AlertRule) SortableFields() []string {
	return []string{"created_at", "updated_at", "name"}
}
