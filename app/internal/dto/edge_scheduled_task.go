package dto

import "github.com/niko-admin/niko-admin/internal/pkg/scopes"

// CreateEdgeScheduledTaskRequest 创建计划任务请求
type CreateEdgeScheduledTaskRequest struct {
	Name             string      `json:"name" binding:"required,max=255"`
	Description      string      `json:"description" binding:"max=500"`
	CronExpr         string      `json:"cron_expr" binding:"required,max=100"`
	TargetType       string      `json:"target_type" binding:"required,oneof=single_node tag"`
	TargetID         string      `json:"target_id" binding:"required"`
	CommandName      string      `json:"command_name" binding:"required,max=100"`
	CommandParams    interface{} `json:"command_params"`
	WaitResponse     bool        `json:"wait_response"`
	WaitTimeoutSec   int         `json:"wait_timeout_sec"`
	MaxRetries       int         `json:"max_retries"`
	RetryIntervalSec int         `json:"retry_interval_sec"`
}

// UpdateEdgeScheduledTaskRequest 更新计划任务请求
type UpdateEdgeScheduledTaskRequest struct {
	Name             string      `json:"name" binding:"required,max=255"`
	Description      string      `json:"description" binding:"max=500"`
	CronExpr         string      `json:"cron_expr" binding:"required,max=100"`
	TargetType       string      `json:"target_type" binding:"required,oneof=single_node tag"`
	TargetID         string      `json:"target_id" binding:"required"`
	CommandName      string      `json:"command_name" binding:"required,max=100"`
	CommandParams    interface{} `json:"command_params"`
	WaitResponse     bool        `json:"wait_response"`
	WaitTimeoutSec   int         `json:"wait_timeout_sec"`
	MaxRetries       int         `json:"max_retries"`
	RetryIntervalSec int         `json:"retry_interval_sec"`
}

// EdgeScheduledTaskListRequest 计划任务分页列表请求
type EdgeScheduledTaskListRequest struct {
	PageRequest
	Keyword string `form:"keyword"`
	Enabled *bool  `form:"enabled"`
}

// FilterScopes 构建筛选条件
func (r *EdgeScheduledTaskListRequest) FilterScopes() []scopes.Scope {
	var sc []scopes.Scope
	if r.Keyword != "" {
		sc = append(sc, scopes.MultiLike([]string{"name", "description"}, r.Keyword))
	}
	if r.Enabled != nil {
		sc = append(sc, scopes.Eq("enabled", *r.Enabled))
	}
	return sc
}

// EdgeScheduledTaskRecordListRequest 执行记录分页列表请求
type EdgeScheduledTaskRecordListRequest struct {
	PageRequest
	TaskID string `form:"task_id"`
	Status string `form:"status"`
}

// FilterScopes 构建筛选条件
func (r *EdgeScheduledTaskRecordListRequest) FilterScopes() []scopes.Scope {
	var sc []scopes.Scope
	if r.TaskID != "" {
		sc = append(sc, scopes.Eq("task_id", r.TaskID))
	}
	if r.Status != "" {
		sc = append(sc, scopes.Eq("status", r.Status))
	}
	return sc
}
