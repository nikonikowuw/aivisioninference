package dto

import (
	"time"

	"github.com/niko-admin/niko-admin/internal/pkg/scopes"
)

// MailConfigRequest is the request body for saving system mail settings.
type MailConfigRequest struct {
	Enabled     bool   `json:"enabled"`
	FromName    string `json:"from_name" binding:"omitempty,max=128"`
	FromAddress string `json:"from_address" binding:"omitempty,email,max=128"`
	ReplyTo     string `json:"reply_to" binding:"omitempty,email,max=128"`

	SMTPEnabled    bool   `json:"smtp_enabled"`
	SMTPHost       string `json:"smtp_host" binding:"omitempty,max=255"`
	SMTPPort       int    `json:"smtp_port" binding:"omitempty,min=1,max=65535"`
	SMTPUsername   string `json:"smtp_username" binding:"omitempty,max=255"`
	SMTPPassword   string `json:"smtp_password" binding:"omitempty,max=512"`
	SMTPEncryption string `json:"smtp_encryption" binding:"omitempty,oneof=none starttls tls"`
	SMTPTimeoutSec int    `json:"smtp_timeout_sec" binding:"omitempty,min=1,max=120"`

	IMAPEnabled     bool   `json:"imap_enabled"`
	IMAPHost        string `json:"imap_host" binding:"omitempty,max=255"`
	IMAPPort        int    `json:"imap_port" binding:"omitempty,min=1,max=65535"`
	IMAPUsername    string `json:"imap_username" binding:"omitempty,max=255"`
	IMAPPassword    string `json:"imap_password" binding:"omitempty,max=512"`
	IMAPEncryption  string `json:"imap_encryption" binding:"omitempty,oneof=none starttls tls"`
	IMAPMailbox     string `json:"imap_mailbox" binding:"omitempty,max=128"`
	IMAPSyncMinutes int    `json:"imap_sync_minutes" binding:"omitempty,min=1,max=1440"`
}

// MailConfigResponse is the sanitized mail configuration.
type MailConfigResponse struct {
	ID                     string `json:"id"`
	Enabled                bool   `json:"enabled"`
	FromName               string `json:"from_name"`
	FromAddress            string `json:"from_address"`
	ReplyTo                string `json:"reply_to"`
	SMTPEnabled            bool   `json:"smtp_enabled"`
	SMTPHost               string `json:"smtp_host"`
	SMTPPort               int    `json:"smtp_port"`
	SMTPUsername           string `json:"smtp_username"`
	SMTPPasswordConfigured bool   `json:"smtp_password_configured"`
	SMTPEncryption         string `json:"smtp_encryption"`
	SMTPTimeoutSec         int    `json:"smtp_timeout_sec"`
	IMAPEnabled            bool   `json:"imap_enabled"`
	IMAPHost               string `json:"imap_host"`
	IMAPPort               int    `json:"imap_port"`
	IMAPUsername           string `json:"imap_username"`
	IMAPPasswordConfigured bool   `json:"imap_password_configured"`
	IMAPEncryption         string `json:"imap_encryption"`
	IMAPMailbox            string `json:"imap_mailbox"`
	IMAPSyncMinutes        int    `json:"imap_sync_minutes"`
	CreatedAt              string `json:"created_at"`
	UpdatedAt              string `json:"updated_at"`
}

// TestSMTPRequest is the request body for sending a test email.
type TestSMTPRequest struct {
	To string `json:"to" binding:"required,email"`
}

// FeedbackCreateRequest is the request body for submitting feedback.
type FeedbackCreateRequest struct {
	Category string `json:"category" binding:"omitempty,max=64"`
	Title    string `json:"title" binding:"required,max=255"`
	Content  string `json:"content" binding:"required,max=5000"`
}

// FeedbackUpdateStatusRequest updates feedback processing status.
type FeedbackUpdateStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=open processing resolved closed"`
}

// FeedbackListRequest is the request for listing feedback.
type FeedbackListRequest struct {
	PageRequest
	Keyword   string     `form:"keyword"`
	Source    string     `form:"source"`
	Status    string     `form:"status"`
	StartTime string     `form:"start_time"`
	EndTime   string     `form:"end_time"`
	FromTime  *time.Time `form:"-" json:"-"`
	ToTime    *time.Time `form:"-" json:"-"`
}

func (r *FeedbackListRequest) FilterScopes() []scopes.Scope {
	var sc []scopes.Scope
	if r.Keyword != "" {
		sc = append(sc, scopes.MultiLike([]string{"title", "content", "email", "category"}, r.Keyword))
	}
	if r.Source != "" {
		sc = append(sc, scopes.Eq("source", r.Source))
	}
	if r.Status != "" {
		sc = append(sc, scopes.Eq("status", r.Status))
	}
	if r.FromTime != nil || r.ToTime != nil {
		sc = append(sc, scopes.TimeRange("created_at", r.FromTime, r.ToTime))
	}
	return sc
}
