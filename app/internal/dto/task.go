// Package dto 定义请求和响应的数据传输结构体，包含参数校验和序列化标签。
package dto

import (
	"time"

	"github.com/niko-admin/niko-admin/internal/pkg/scopes"
)

// CreateTaskRequest is the request body for creating a task.
type CreateTaskRequest struct {
	Type    string `json:"type" binding:"required"`
	Payload string `json:"payload"`
}

// TaskResponse is the task data returned in API responses.
type TaskResponse struct {
	ID           string  `json:"id"`
	TaskID       string  `json:"task_id"`
	Type         string  `json:"type"`
	Payload      string  `json:"payload"`
	Status       string  `json:"status"`
	RetryCount   int     `json:"retry_count"`
	MaxRetries   int     `json:"max_retries"`
	Result       string  `json:"result"`
	ErrorMessage string  `json:"error_message"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
	FinishedAt   *string `json:"finished_at"`
}

// TaskListRequest is the request for listing tasks with filters.
type TaskListRequest struct {
	PageRequest
	Keyword   string     `form:"keyword"`
	Type      string     `form:"type"`
	Status    string     `form:"status"`
	StartTime string     `form:"start_time"`
	EndTime   string     `form:"end_time"`
	FromTime  *time.Time `form:"-" json:"-"`
	ToTime    *time.Time `form:"-" json:"-"`
}

// FilterScopes 返回当前请求对应的 GORM 查询范围函数列表，支持关键词、类型、状态和时间范围过滤。
func (r *TaskListRequest) FilterScopes() []scopes.Scope {
	var sc []scopes.Scope
	if r.Keyword != "" {
		sc = append(sc, scopes.MultiLike([]string{"type", "task_id"}, r.Keyword))
	}
	if r.Type != "" {
		sc = append(sc, scopes.Eq("type", r.Type))
	}
	if r.Status != "" {
		sc = append(sc, scopes.Eq("status", r.Status))
	}
	if r.FromTime != nil || r.ToTime != nil {
		sc = append(sc, scopes.TimeRange("created_at", r.FromTime, r.ToTime))
	}
	return sc
}
