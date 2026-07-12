package model

import (
	"time"

	"gorm.io/datatypes"
)

// 计划任务执行记录状态常量
const (
	ScheduledTaskRecordStatusPending    = "pending"
	ScheduledTaskRecordStatusDispatched = "dispatched"
	ScheduledTaskRecordStatusSuccess    = "success"
	ScheduledTaskRecordStatusFailed     = "failed"
)

// EdgeScheduledTaskRecord 计划任务执行记录
type EdgeScheduledTaskRecord struct {
	BaseModel
	TaskID          string         `gorm:"type:uuid;not null;index:idx_scheduled_task_records_task_id;comment:任务ID" json:"task_id"`
	NodeID          string         `gorm:"type:uuid;not null;index:idx_scheduled_task_records_node_id;comment:目标节点ID" json:"node_id"`
	Status          string         `gorm:"type:varchar(20);not null;default:pending;index;comment:状态(pending/dispatched/success/failed)" json:"status"`
	TraceID         string         `gorm:"type:varchar(50);comment:MQTT trace_id" json:"trace_id,omitempty"`
	RequestPayload  datatypes.JSON `gorm:"type:jsonb;comment:下发请求参数" json:"request_payload,omitempty" swaggertype:"object"`
	ResponsePayload datatypes.JSON `gorm:"type:jsonb;comment:响应内容" json:"response_payload,omitempty" swaggertype:"object"`
	ErrorMessage    string         `gorm:"type:text;comment:错误信息" json:"error_message,omitempty"`
	RetryCount      int            `gorm:"default:0;comment:已重试次数" json:"retry_count"`
	ExecutedAt      *time.Time     `gorm:"comment:执行时间" json:"executed_at,omitempty"`
	CompletedAt     *time.Time     `gorm:"comment:完成时间" json:"completed_at,omitempty"`
}

// TableName 指定表名
func (EdgeScheduledTaskRecord) TableName() string {
	return "edge_scheduled_task_records"
}

// SortableFields 返回允许排序的字段列表
func (EdgeScheduledTaskRecord) SortableFields() []string {
	return []string{"created_at", "executed_at", "status"}
}
