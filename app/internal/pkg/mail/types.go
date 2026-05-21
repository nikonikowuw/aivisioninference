// Package mail provides SMTP delivery and IMAP synchronization primitives.
package mail

import "time"

const (
	EncryptionNone     = "none"
	EncryptionTLS      = "tls"
	EncryptionSTARTTLS = "starttls"
)

// SMTPConfig contains connection settings for SMTP delivery.
type SMTPConfig struct {
	Enabled     bool
	Host        string
	Port        int
	Username    string
	Password    string
	Encryption  string
	Timeout     time.Duration
	FromName    string
	FromAddress string
	ReplyTo     string
}

// Message describes an email message to send.
type Message struct {
	To       []string
	Subject  string
	TextBody string
	HTMLBody string
	FromName string
	From     string
	ReplyTo  string
}

// IMAPConfig contains connection settings for mailbox synchronization.
type IMAPConfig struct {
	Enabled    bool
	Host       string
	Port       int
	Username   string
	Password   string
	Encryption string
	Mailbox    string
	Timeout    time.Duration
}

// InboundMessage describes a parsed email fetched from IMAP.
type InboundMessage struct {
	UID       uint32
	MessageID string
	From      string
	To        string
	Subject   string
	TextBody  string
	HTMLBody  string
	Summary   string
	Date      *time.Time
	RawSize   int64
}
