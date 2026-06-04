// Package model 定义系统数据模型，包含 GORM 结构体和数据库表映射。
package model

import (
	"time"

	"gorm.io/datatypes"
)

// Webhook 常量定义
const (
	WebhookPushEventAlarm        = "alarm"
	WebhookPushEventRecognition  = "recognition"
	WebhookPushEventCapture      = "capture"

	WebhookPushStatusPending  = "pending"
	WebhookPushStatusSuccess  = "success"
	WebhookPushStatusFailed   = "failed"
	WebhookPushStatusRetrying = "retrying"
)

// WebhookConfig 表示 Webhook 告警推送配置
type WebhookConfig struct {
	BaseModel
	Name              string         `gorm:"type:varchar(128);not null" json:"name"`
	URL               string         `gorm:"type:text;not null" json:"url"`
	Headers           datatypes.JSON `gorm:"type:jsonb;default:'{}'" json:"headers"`
	PushAlarm         bool           `gorm:"default:true" json:"push_alarm"`
	PushRecognition   bool           `gorm:"default:false" json:"push_recognition"`
	PushCapture       bool           `gorm:"default:false" json:"push_capture"`
	MaxRetries        int            `gorm:"default:3" json:"max_retries"`
	RetryIntervalBase int            `gorm:"default:5" json:"retry_interval_base"`
	AlarmSuppressSec  int            `gorm:"default:300" json:"alarm_suppress_sec"`
	NotifyUsers       datatypes.JSON `gorm:"type:jsonb;default:'[]'" json:"notify_users,omitempty"`
	RateLimit         *int           `json:"rate_limit,omitempty"`
	Enabled           bool           `gorm:"default:true;index" json:"enabled"`
}

// SortableFields 返回允许排序的字段列表
func (WebhookConfig) SortableFields() []string {
	return []string{"created_at", "name"}
}

// TableName 覆盖 WebhookConfig 的默认表名
func (WebhookConfig) TableName() string {
	return "webhook_configs"
}

// WebhookPushLog 表示 Webhook 推送记录，支持指数退避重试和事件关联
type WebhookPushLog struct {
	BaseModel
	WebhookID     string         `gorm:"type:uuid;not null;index" json:"webhook_id"`
	EventID       string         `gorm:"type:uuid;not null" json:"event_id"`
	EventType     string         `gorm:"type:varchar(16);not null;check:event_type IN ('alarm','recognition','capture')" json:"event_type"`
	CorrelationID *string        `gorm:"type:uuid;index" json:"correlation_id,omitempty"`
	RequestURL    string         `gorm:"type:text;not null" json:"request_url"`
	RequestBody   datatypes.JSON `gorm:"type:jsonb" json:"request_body,omitempty"`
	ResponseStatus int           `json:"response_status,omitempty"`
	ResponseBody  string         `gorm:"type:text" json:"response_body,omitempty"`
	Attempt       int            `gorm:"default:1" json:"attempt"`
	MaxAttempts   int            `gorm:"default:3" json:"max_attempts"`
	NextRetryAt   *time.Time     `json:"next_retry_at,omitempty"`
	Status        string         `gorm:"type:varchar(16);not null;default:pending;index" json:"status"`
	ErrorMessage  string         `gorm:"type:text" json:"error_message,omitempty"`
	CompletedAt   *time.Time     `json:"completed_at,omitempty"`
}

// SortableFields 返回允许排序的字段列表
func (WebhookPushLog) SortableFields() []string {
	return []string{"created_at", "status", "attempt"}
}

// TableName 覆盖 WebhookPushLog 的默认表名
func (WebhookPushLog) TableName() string {
	return "webhook_push_logs"
}
