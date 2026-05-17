package dto

import (
	"time"

	"github.com/niko-admin/niko-admin/internal/pkg/scopes"
)

const DateTimeFormat = time.RFC3339

// AuditLogResponse is the audit log data returned in API responses.
type AuditLogResponse struct {
	ID             string    `json:"id"`
	UserID         *string   `json:"user_id"`
	Username       string    `json:"username"`
	Action         string    `json:"action"`
	ResourceType   string    `json:"resource_type"`
	ResourceID     string    `json:"resource_id"`
	RequestPath    string    `json:"request_path"`
	RequestMethod  string    `json:"request_method"`
	RequestIP      string    `json:"request_ip"`
	UserAgent      string    `json:"user_agent"`
	ResponseStatus int       `json:"response_status"`
	DurationMs     int64     `json:"duration_ms"`
	ResultSummary  string    `json:"result_summary"`
	ErrorSummary   string    `json:"error_summary"`
	CreatedAt      time.Time `json:"created_at"`
}

// ListAuditLogRequest is the request for listing audit logs with filters.
type ListAuditLogRequest struct {
	PageRequest
	Keyword      string     `form:"keyword"`
	Action       string     `form:"action"`
	ResourceType string     `form:"resource_type"`
	StartTime    string     `form:"start_time"`
	EndTime      string     `form:"end_time"`
	FromTime     *time.Time `form:"-" json:"-"` // parsed by service, used by FilterScopes
	ToTime       *time.Time `form:"-" json:"-"` // parsed by service, used by FilterScopes
}

func (r *ListAuditLogRequest) FilterScopes() []scopes.Scope {
	var sc []scopes.Scope
	if r.Keyword != "" {
		sc = append(sc, scopes.MultiLike([]string{"username", "action", "resource_type", "request_path"}, r.Keyword))
	}
	if r.Action != "" {
		sc = append(sc, scopes.Eq("action", r.Action))
	}
	if r.ResourceType != "" {
		sc = append(sc, scopes.Eq("resource_type", r.ResourceType))
	}
	if r.FromTime != nil || r.ToTime != nil {
		sc = append(sc, scopes.TimeRange("created_at", r.FromTime, r.ToTime))
	}
	return sc
}
