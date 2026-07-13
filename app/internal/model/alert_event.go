package model

import (
	"time"

	"gorm.io/gorm"
)

// AlertEvent 告警事件模型
// 记录每次触发的告警及其生命周期（触发→恢复→确认）
type AlertEvent struct {
	BaseModel
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	RuleID      string  `gorm:"type:uuid;not null;index;comment:关联告警规则ID" json:"rule_id"`
	NodeID      string  `gorm:"type:uuid;not null;index;comment:关联边缘节点ID" json:"node_id"`
	MetricValue float64 `gorm:"not null;comment:触发时的指标值" json:"metric_value"`
	Status      string  `gorm:"type:varchar(20);not null;default:'firing';comment:状态(firing/resolved/acknowledged)" json:"status"`
	FiredAt     time.Time `gorm:"not null;default:now();comment:触发时间" json:"fired_at"`
	ResolvedAt  *time.Time `gorm:"comment:恢复时间" json:"resolved_at,omitempty"`
	AcknowledgedBy *string `gorm:"type:varchar(255);comment:确认人" json:"acknowledged_by,omitempty"`
	AcknowledgedAt *time.Time `gorm:"comment:确认时间" json:"acknowledged_at,omitempty"`
	NotifySent    bool   `gorm:"not null;default:false;comment:是否已发送通知" json:"notify_sent"`
	NotifySentAt  *time.Time `gorm:"comment:通知发送时间" json:"notify_sent_at,omitempty"`
}

// AlertEvent 状态常量
const (
	AlertEventStatusFiring        = "firing"
	AlertEventStatusResolved      = "resolved"
	AlertEventStatusAcknowledged  = "acknowledged"
)

// TableName 指定表名
func (AlertEvent) TableName() string {
	return "alert_events"
}

// SortableFields 返回允许排序的字段列表
func (AlertEvent) SortableFields() []string {
	return []string{"created_at", "fired_at", "resolved_at"}
}
