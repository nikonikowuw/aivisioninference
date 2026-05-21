package model

import "time"

const (
	MailEncryptionNone     = "none"
	MailEncryptionTLS      = "tls"
	MailEncryptionSTARTTLS = "starttls"

	EmailTokenPurposeRegisterVerify = "register_email_verify"
	EmailTokenPurposeChangeVerify   = "change_email_verify"
	EmailTokenPurposePasswordReset  = "password_reset"

	InboundEmailStatusNew       = "new"
	InboundEmailStatusProcessed = "processed"
)

// MailConfig 存储系统级 SMTP 和 IMAP 配置
type MailConfig struct {
	BaseModel
	Enabled     bool   `gorm:"default:false" json:"enabled"`
	FromName    string `gorm:"type:varchar(128)" json:"from_name"`
	FromAddress string `gorm:"type:varchar(128)" json:"from_address"`
	ReplyTo     string `gorm:"type:varchar(128)" json:"reply_to"`

	SMTPEnabled    bool   `gorm:"default:false" json:"smtp_enabled"`
	SMTPHost       string `gorm:"type:varchar(255)" json:"smtp_host"`
	SMTPPort       int    `json:"smtp_port"`
	SMTPUsername   string `gorm:"type:varchar(255)" json:"smtp_username"`
	SMTPPassword   string `gorm:"type:text" json:"-"`
	SMTPEncryption string `gorm:"type:varchar(20);default:'starttls'" json:"smtp_encryption"`
	SMTPTimeoutSec int    `gorm:"default:10" json:"smtp_timeout_sec"`

	IMAPEnabled     bool   `gorm:"default:false" json:"imap_enabled"`
	IMAPHost        string `gorm:"type:varchar(255)" json:"imap_host"`
	IMAPPort        int    `json:"imap_port"`
	IMAPUsername    string `gorm:"type:varchar(255)" json:"imap_username"`
	IMAPPassword    string `gorm:"type:text" json:"-"`
	IMAPEncryption  string `gorm:"type:varchar(20);default:'tls'" json:"imap_encryption"`
	IMAPMailbox     string `gorm:"type:varchar(128);default:'INBOX'" json:"imap_mailbox"`
	IMAPSyncMinutes int    `gorm:"default:10" json:"imap_sync_minutes"`
}

// SortableFields 返回允许排序的字段列表
func (MailConfig) SortableFields() []string {
	return []string{"created_at", "updated_at"}
}

// EmailToken 存储一次性邮箱验证和密码重置令牌
type EmailToken struct {
	BaseModel
	UserID       *string    `gorm:"type:uuid;index" json:"user_id"`
	Email        string     `gorm:"type:varchar(128);index;not null" json:"email"`
	Purpose      string     `gorm:"type:varchar(40);index;not null" json:"purpose"`
	TokenHash    string     `gorm:"type:varchar(128);uniqueIndex;not null" json:"-"`
	ExpiresAt    time.Time  `gorm:"index" json:"expires_at"`
	UsedAt       *time.Time `json:"used_at"`
	RequestIP    string     `gorm:"type:varchar(45)" json:"request_ip"`
	AttemptCount int        `gorm:"default:0" json:"attempt_count"`
}

// SortableFields 返回允许排序的字段列表
func (EmailToken) SortableFields() []string {
	return []string{"created_at", "expires_at"}
}

// InboundEmail 存储从配置的 IMAP 邮箱同步的邮件
type InboundEmail struct {
	BaseModel
	Account        string     `gorm:"type:varchar(255);not null;index:idx_inbound_email_unique,priority:1" json:"account"`
	Mailbox        string     `gorm:"type:varchar(128);not null;index:idx_inbound_email_unique,priority:2" json:"mailbox"`
	UID            uint32     `gorm:"index:idx_inbound_email_unique,priority:3" json:"uid"`
	MessageID      string     `gorm:"type:varchar(255);index" json:"message_id"`
	From           string     `gorm:"type:varchar(512)" json:"from"`
	To             string     `gorm:"type:varchar(1024)" json:"to"`
	Subject        string     `gorm:"type:varchar(512)" json:"subject"`
	TextBody       string     `gorm:"type:text" json:"text_body"`
	HTMLBody       string     `gorm:"type:text" json:"html_body"`
	Summary        string     `gorm:"type:varchar(512)" json:"summary"`
	EmailDate      *time.Time `json:"email_date"`
	Status         string     `gorm:"type:varchar(20);default:'new'" json:"status"`
	ProcessedAt    *time.Time `json:"processed_at"`
	RawSize        int64      `json:"raw_size"`
	AttachmentInfo string     `gorm:"type:text" json:"attachment_info"`
}

// SortableFields 返回允许排序的字段列表
func (InboundEmail) SortableFields() []string {
	return []string{"created_at", "email_date", "status"}
}
