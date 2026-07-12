package model

import "time"

// EdgeNodeTaskExecution 状态常量
const (
	TaskExecRunning = "running"
	TaskExecSuccess = "success"
	TaskExecFailed  = "failed"
	TaskExecTimeout = "timeout"
)

// EdgeNodeTaskExecution 边缘节点任务执行记录（Phase 3 — 远程运维）
type EdgeNodeTaskExecution struct {
	BaseModel
	TaskID     string     `gorm:"type:char(36);not null;index:idx_taskexec_task;comment:所属任务ID" json:"task_id"`
	Status     string     `gorm:"type:varchar(20);not null;default:'running';comment:执行状态(running/success/failed/timeout)" json:"status"`
	Stdout     string     `gorm:"type:text;comment:标准输出" json:"stdout,omitempty"`
	Stderr     string     `gorm:"type:text;comment:标准错误" json:"stderr,omitempty"`
	ExitCode   *int       `gorm:"comment:退出码" json:"exit_code,omitempty"`
	StartedAt  *time.Time `gorm:"index;comment:开始时间" json:"started_at,omitempty"`
	FinishedAt *time.Time `gorm:"comment:结束时间" json:"finished_at,omitempty"`
}

// TableName 指定表名
func (EdgeNodeTaskExecution) TableName() string {
	return "edge_node_task_executions"
}

// SortableFields 返回允许排序的字段列表
func (EdgeNodeTaskExecution) SortableFields() []string {
	return []string{"created_at", "started_at", "finished_at"}
}
