package model

import "time"

// 邮件配置相关的加密类型与邮箱令牌用途等常量定义
const (
	MailEncryptionNone     = "none"     // 不加密
	MailEncryptionTLS      = "tls"      // TLS 加密方式
	MailEncryptionSTARTTLS = "starttls" // STARTTLS 升级加密方式

	EmailTokenPurposeRegisterVerify = "register_email_verify" // 注册账号邮箱验证
	EmailTokenPurposeChangeVerify   = "change_email_verify"   // 绑定邮箱变更验证
	EmailTokenPurposePasswordReset  = "password_reset"        // 重置密码邮箱验证

	InboundEmailStatusNew       = "new"       // 新同步的收件
	InboundEmailStatusProcessed = "processed" // 已处理的收件
)

// MailConfig 存储系统级 SMTP 和 IMAP 配置
type MailConfig struct {
	BaseModel
	Enabled     bool   `gorm:"default:false" json:"enabled"`          // 是否启用邮件模块
	FromName    string `gorm:"type:varchar(128)" json:"from_name"`    // 发件人显示名称
	FromAddress string `gorm:"type:varchar(128)" json:"from_address"` // 发件人邮箱地址
	ReplyTo     string `gorm:"type:varchar(128)" json:"reply_to"`     // 回信地址

	SMTPEnabled    bool   `gorm:"default:false" json:"smtp_enabled"`                          // 是否启用发件 SMTP 服务
	SMTPHost       string `gorm:"type:varchar(255)" json:"smtp_host"`                         // SMTP 服务器地址
	SMTPPort       int    `json:"smtp_port"`                                                  // SMTP 端口号
	SMTPUsername   string `gorm:"type:varchar(255)" json:"smtp_username"`                     // SMTP 登录用户名
	SMTPPassword   string `gorm:"type:text" json:"-"`                                         // SMTP 登录密码 (敏感字段，不输出 json)
	SMTPEncryption string `gorm:"type:varchar(20);default:'starttls'" json:"smtp_encryption"` // SMTP 加密方式 (none/tls/starttls)
	SMTPTimeoutSec int    `gorm:"default:10" json:"smtp_timeout_sec"`                         // SMTP 连接超时秒数

	IMAPEnabled     bool   `gorm:"default:false" json:"imap_enabled"`                     // 是否启用收件 IMAP 同步
	IMAPHost        string `gorm:"type:varchar(255)" json:"imap_host"`                    // IMAP 服务器地址
	IMAPPort        int    `json:"imap_port"`                                             // IMAP 端口号
	IMAPUsername    string `gorm:"type:varchar(255)" json:"imap_username"`                // IMAP 登录用户名
	IMAPPassword    string `gorm:"type:text" json:"-"`                                    // IMAP 登录密码 (敏感字段，不输出 json)
	IMAPEncryption  string `gorm:"type:varchar(20);default:'tls'" json:"imap_encryption"` // IMAP 加密方式 (none/tls/starttls)
	IMAPMailbox     string `gorm:"type:varchar(128);default:'INBOX'" json:"imap_mailbox"` // IMAP 目标邮件箱名 (默认 INBOX)
	IMAPSyncMinutes int    `gorm:"default:10" json:"imap_sync_minutes"`                   // 自动同步间隔分钟数
}

// SortableFields 返回允许排序的字段列表
func (MailConfig) SortableFields() []string {
	return []string{"created_at", "updated_at"}
}

// EmailToken 存储一次性邮箱验证和密码重置令牌
type EmailToken struct {
	BaseModel
	UserID       *string    `gorm:"type:uuid;index" json:"user_id"`                  // 关联的用户 ID
	Email        string     `gorm:"type:varchar(128);index;not null" json:"email"`   // 验证的目标邮箱
	Purpose      string     `gorm:"type:varchar(40);index;not null" json:"purpose"`  // 令牌用途类型
	TokenHash    string     `gorm:"type:varchar(128);uniqueIndex;not null" json:"-"` // 令牌散列值，确保数据库中的安全性 (不输出 json)
	ExpiresAt    time.Time  `gorm:"index" json:"expires_at"`                         // 过期时间
	UsedAt       *time.Time `json:"used_at"`                                         // 被使用的时间 (为空表示未使用)
	RequestIP    string     `gorm:"type:varchar(45)" json:"request_ip"`              // 发起申请的客户端 IP
	AttemptCount int        `gorm:"default:0" json:"attempt_count"`                  // 验证尝试次数限制计数
}

// SortableFields 返回允许排序的字段列表
func (EmailToken) SortableFields() []string {
	return []string{"created_at", "expires_at"}
}

// InboundEmail 存储从配置的 IMAP 邮箱同步下来的入站邮件
type InboundEmail struct {
	BaseModel
	Account        string     `gorm:"type:varchar(255);not null;index:idx_inbound_email_unique,priority:1" json:"account"` // 收信账号
	Mailbox        string     `gorm:"type:varchar(128);not null;index:idx_inbound_email_unique,priority:2" json:"mailbox"` // 邮箱箱名
	UID            uint32     `gorm:"index:idx_inbound_email_unique,priority:3" json:"uid"`                                // IMAP 服务器上的唯一 UID
	MessageID      string     `gorm:"type:varchar(255);index" json:"message_id"`                                           // 邮件标准 Message-ID
	From           string     `gorm:"type:varchar(512)" json:"from"`                                                       // 发件人信息 (如 display_name <email>)
	To             string     `gorm:"type:varchar(1024)" json:"to"`                                                        // 收件人信息列表
	Subject        string     `gorm:"type:varchar(512)" json:"subject"`                                                    // 邮件主题
	TextBody       string     `gorm:"type:text" json:"text_body"`                                                          // 纯文本正文
	HTMLBody       string     `gorm:"type:text" json:"html_body"`                                                          // HTML正文
	Summary        string     `gorm:"type:varchar(512)" json:"summary"`                                                    // 邮件摘要
	EmailDate      *time.Time `json:"email_date"`                                                                          // 邮件头部声明的发送时间
	Status         string     `gorm:"type:varchar(20);default:'new'" json:"status"`                                        // 同步处理状态 (new=未处理, processed=已处理)
	ProcessedAt    *time.Time `json:"processed_at"`                                                                        // 业务逻辑处理时间
	RawSize        int64      `json:"raw_size"`                                                                            // 邮件原始字节大小
	AttachmentInfo string     `gorm:"type:text" json:"attachment_info"`                                                    // 附件元数据 JSON 信息 (如文件名、大小列表)
}

// SortableFields 返回允许排序的字段列表
func (InboundEmail) SortableFields() []string {
	return []string{"created_at", "email_date", "status"}
}
