package dto

import "time"

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
	ResponseStatus int       `json:"response_status"`
	DurationMs     int64     `json:"duration_ms"`
	CreatedAt      time.Time `json:"created_at"`
}

// ListAuditLogRequest is the request for listing audit logs with filters.
type ListAuditLogRequest struct {
	PageRequest
	UserID       *string `form:"user_id"`
	Action       string  `form:"action"`
	ResourceType string  `form:"resource_type"`
	StartTime    string  `form:"start_time"`
	EndTime      string  `form:"end_time"`
}
