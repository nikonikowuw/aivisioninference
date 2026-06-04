// Package model 定义系统数据模型，包含 GORM 结构体和数据库表映射。
package model

import (
	"time"

	"gorm.io/datatypes"
)

// AI 异步任务常量定义
const (
	AIAsyncTaskStatusQueued    = "queued"
	AIAsyncTaskStatusRunning   = "running"
	AIAsyncTaskStatusCompleted = "completed"
	AIAsyncTaskStatusFailed    = "failed"
	AIAsyncTaskStatusCancelled = "cancelled"

	AIAsyncTaskTypeEmbedding     = "extract_embedding"
	AIAsyncTaskTypeWebhookPush   = "webhook_push"
	AIAsyncTaskTypeCleanup       = "storage_cleanup"
)

// AIAsyncTask 表示 AIVisionInference 领域的异步任务元数据
// 补充 scaffold model.Task（通用后台任务），用于特征提取、Webhook 重试、存储清理等
type AIAsyncTask struct {
	BaseModel
	TaskType    string         `gorm:"type:varchar(64);not null;index" json:"task_type"`
	Payload     datatypes.JSON `gorm:"type:jsonb;not null" json:"payload"`
	Status      string         `gorm:"type:varchar(16);not null;default:queued;index" json:"status"`
	Progress    *float64       `gorm:"type:numeric(5,2)" json:"progress,omitempty"`
	RetryCount  int            `gorm:"default:0" json:"retry_count"`
	MaxRetries  int            `gorm:"default:3" json:"max_retries"`
	LastError   string         `gorm:"type:text" json:"last_error,omitempty"`
	ScheduledAt *time.Time     `json:"scheduled_at,omitempty"`
	StartedAt   *time.Time     `json:"started_at,omitempty"`
	CompletedAt *time.Time     `json:"completed_at,omitempty"`
}

// SortableFields 返回允许排序的字段列表
func (AIAsyncTask) SortableFields() []string {
	return []string{"created_at", "task_type", "status"}
}

// TableName 覆盖 AIAsyncTask 的默认表名
func (AIAsyncTask) TableName() string {
	return "ai_async_tasks"
}
