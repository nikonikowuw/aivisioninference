// Package dto 定义请求和响应的数据传输结构体，包含参数校验和序列化标签。
package dto

import (
	"time"

	"github.com/niko-admin/niko-admin/internal/pkg/scopes"
)

// DateTimeFormat 是审计日志等响应中使用的时间戳格式（RFC3339）。
const DateTimeFormat = time.RFC3339

// AuditLogResponse is the audit log data returned in API responses.
type AuditLogResponse struct {
	ID                 string    `json:"id"`
	UserID             *string   `json:"user_id"`
	Username           string    `json:"username"`
	ActionType         string    `json:"action_type"`         // 操作类型原始 key，用于筛选
	ActionTypeLabel    string    `json:"action_type_label"`   // 操作类型翻译文本，用于展示
	ResourceType       string    `json:"resource_type"`       // 资源类型原始值，用于筛选
	ResourceTypeLabel  string    `json:"resource_type_label"` // 资源类型翻译文本，用于展示
	ResourceID         string    `json:"resource_id"`
	RequestPath        string    `json:"request_path"`
	RequestMethod      string    `json:"request_method"`       // HTTP 方法原始值，用于筛选
	RequestMethodLabel string    `json:"request_method_label"` // HTTP 方法翻译文本，用于展示
	RequestIP          string    `json:"request_ip"`
	UserAgent          string    `json:"user_agent"`
	ResponseStatus     int       `json:"response_status"`
	DurationMs         int64     `json:"duration_ms"`
	ResultSummary      string    `json:"result_summary"`       // 结果摘要原始值，用于筛选
	ResultSummaryLabel string    `json:"result_summary_label"` // 结果摘要翻译文本，用于展示
	CreatedAt          time.Time `json:"created_at"`
}

// ListAuditLogRequest is the request for listing audit logs with filters.
type ListAuditLogRequest struct {
	PageRequest
	Keyword      string     `form:"keyword"`
	ResourceType string     `form:"resource_type"`
	ActionType   string     `form:"action_type"`
	Result       string     `form:"result"`
	StartTime    string     `form:"start_time"`
	EndTime      string     `form:"end_time"`
	FromTime     *time.Time `form:"-" json:"-"` // parsed by service, used by FilterScopes
	ToTime       *time.Time `form:"-" json:"-"` // parsed by service, used by FilterScopes
}

// FilterScopes 返回审计日志列表的过滤条件，支持关键词、资源、操作、结果和时间范围。
func (r *ListAuditLogRequest) FilterScopes() []scopes.Scope {
	var sc []scopes.Scope
	if r.Keyword != "" {
		sc = append(sc, scopes.MultiLike([]string{"username", "resource_type", "request_path", "action_type"}, r.Keyword))
	}
	if r.ResourceType != "" {
		sc = append(sc, scopes.Eq("resource_type", r.ResourceType))
	}
	if r.ActionType != "" {
		sc = append(sc, scopes.Like("action_type", r.ActionType))
	}
	if r.Result != "" {
		sc = append(sc, scopes.Eq("result_summary", r.Result))
	}
	if r.FromTime != nil || r.ToTime != nil {
		sc = append(sc, scopes.TimeRange("created_at", r.FromTime, r.ToTime))
	}
	return sc
}
