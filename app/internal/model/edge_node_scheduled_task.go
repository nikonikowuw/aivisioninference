package model

import "time"

// EdgeNodeScheduledTask 状态常量
const (
	ScheduledTaskActive   = "active"
	ScheduledTaskDisabled = "disabled"
	ScheduledTaskRunning  = "running"
)

// EdgeNodeScheduledTask 边缘节点定时任务模型（Phase 3 — 远程运维）
type EdgeNodeScheduledTask struct {
	BaseModel
	NodeID         string     `gorm:"type:char(36);not null;index:idx_schedtask_node;comment:所属节点ID" json:"node_id"`
	Name           string     `gorm:"type:varchar(255);not null;comment:任务名称" json:"name"`
	Command        string     `gorm:"type:text;not null;comment:Shell命令" json:"command"`
	CronExpr       string     `gorm:"type:varchar(100);comment:Cron表达式，NULL表示一次性任务" json:"cron_expr,omitempty"`
	Status         string     `gorm:"type:varchar(20);not null;default:'active';comment:状态(active/disabled/running)" json:"status"`
	TimeoutSeconds int        `gorm:"not null;default:30;comment:超时时间(秒)" json:"timeout_seconds"`
	LastRunAt      *time.Time `gorm:"index;comment:最后执行时间" json:"last_run_at,omitempty"`
}

// TableName 指定表名
func (EdgeNodeScheduledTask) TableName() string {
	return "edge_node_scheduled_tasks"
}

// SortableFields 返回允许排序的字段列表
func (EdgeNodeScheduledTask) SortableFields() []string {
	return []string{"created_at", "updated_at", "last_run_at"}
}
