// Package mail 提供 SMTP 邮件发送和 IMAP 邮件同步功能。
package mail

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"mime"
	"net"
	"net/mail"
	"net/smtp"
	"strings"
	"time"
)

// SMTPClient sends email through a configured SMTP server.
type SMTPClient struct {
	cfg SMTPConfig
}

// NewSMTPClient creates a SMTPClient.
func NewSMTPClient(cfg SMTPConfig) *SMTPClient {
	if cfg.Timeout <= 0 {
		cfg.Timeout = 10 * time.Second
	}
	return &SMTPClient{cfg: cfg}
}

// Send sends the message through SMTP.
func (c *SMTPClient) Send(ctx context.Context, msg Message) error {
	if !c.cfg.Enabled {
		return errors.New("smtp is disabled")
	}
	if c.cfg.Host == "" || c.cfg.Port == 0 {
		return errors.New("smtp host or port is empty")
	}

	from := firstNonEmpty(msg.From, c.cfg.FromAddress, c.cfg.Username)
	if from == "" {
		return errors.New("smtp sender is empty")
	}
	if len(msg.To) == 0 {
		return errors.New("smtp recipients are empty")
	}

	addr := fmt.Sprintf("%s:%d", c.cfg.Host, c.cfg.Port)
	// 根据加密配置选择直连 TLS（SSL）或明文 TCP 连接，STARTTLS 在连接升级阶段处理。
	conn, err := dialContext(ctx, addr, c.cfg.Timeout, strings.EqualFold(c.cfg.Encryption, EncryptionTLS), c.cfg.Host)
	if err != nil {
		return fmt.Errorf("connect smtp server: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, c.cfg.Host)
	if err != nil {
		return fmt.Errorf("create smtp client: %w", err)
	}
	defer client.Close()

	// STARTTLS 加密：通过明文连接建立后，使用 STARTTLS 命令升级到 TLS 加密通道。
	if strings.EqualFold(c.cfg.Encryption, EncryptionSTARTTLS) {
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err := client.StartTLS(&tls.Config{ServerName: c.cfg.Host, MinVersion: tls.VersionTLS12}); err != nil {
				return fmt.Errorf("start tls: %w", err)
			}
		}
	}

	// SMTP AUTH：使用 PLAIN 机制认证，密码在 TLS 加密通道内传输，否则明文暴露。
	if c.cfg.Username != "" {
		if err := client.Auth(smtp.PlainAuth("", c.cfg.Username, c.cfg.Password, c.cfg.Host)); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}

	if err := client.Mail(from); err != nil {
		return fmt.Errorf("set sender: %w", err)
	}
	// 遍历所有收件人逐一发送 RCPT 命令，跳过空地址避免 SMTP 协议错误。
	for _, to := range msg.To {
		if strings.TrimSpace(to) == "" {
			continue
		}
		if err := client.Rcpt(to); err != nil {
			return fmt.Errorf("set recipient: %w", err)
		}
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("open smtp data: %w", err)
	}
	if _, err := w.Write(buildMIMEMessage(c.cfg, msg, from)); err != nil {
		_ = w.Close()
		return fmt.Errorf("write smtp data: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("close smtp data: %w", err)
	}
	return client.Quit()
}

func dialContext(ctx context.Context, addr string, timeout time.Duration, withTLS bool, serverName string) (net.Conn, error) {
	dialer := &net.Dialer{Timeout: timeout}
	if deadline, ok := ctx.Deadline(); ok {
		dialer.Deadline = deadline
	}
	if withTLS {
		return tls.DialWithDialer(dialer, "tcp", addr, &tls.Config{ServerName: serverName, MinVersion: tls.VersionTLS12})
	}
	return dialer.DialContext(ctx, "tcp", addr)
}

// buildMIMEMessage 构造符合 RFC 2822 的 MIME 邮件。
// 同时存在文本和 HTML 时使用 multipart/alternative 格式，客户端可根据能力选择渲染。
// Subject 使用 Q 编码处理非 ASCII 字符，确保邮件客户端正确显示中文标题。
func buildMIMEMessage(cfg SMTPConfig, msg Message, from string) []byte {
	var buf bytes.Buffer
	fromName := firstNonEmpty(msg.FromName, cfg.FromName)
	fromHeader := from
	if fromName != "" {
		fromHeader = (&mail.Address{Name: fromName, Address: from}).String()
	}

	writeHeader(&buf, "From", fromHeader)
	writeHeader(&buf, "To", strings.Join(msg.To, ", "))
	writeHeader(&buf, "Subject", mime.QEncoding.Encode("utf-8", msg.Subject))
	writeHeader(&buf, "MIME-Version", "1.0")
	if replyTo := firstNonEmpty(msg.ReplyTo, cfg.ReplyTo); replyTo != "" {
		writeHeader(&buf, "Reply-To", replyTo)
	}

	if msg.TextBody != "" && msg.HTMLBody != "" {
		// multipart/alternative 格式：纯文本在前、HTML 在后，客户端优先显示后者。
		boundary := fmt.Sprintf("niko-%d", time.Now().UnixNano())
		writeHeader(&buf, "Content-Type", `multipart/alternative; boundary="`+boundary+`"`)
		buf.WriteString("\r\n")
		writePart(&buf, boundary, "text/plain; charset=utf-8", msg.TextBody)
		writePart(&buf, boundary, "text/html; charset=utf-8", msg.HTMLBody)
		buf.WriteString("--" + boundary + "--\r\n")
		return buf.Bytes()
	}

	// 单种内容格式：直接输出，根据可用内容选择 text/plain 或 text/html。
	contentType := "text/plain; charset=utf-8"
	body := msg.TextBody
	if msg.HTMLBody != "" {
		contentType = "text/html; charset=utf-8"
		body = msg.HTMLBody
	}
	writeHeader(&buf, "Content-Type", contentType)
	// 使用 8bit 传输编码而非 7bit：现代 SMTP 服务器普遍支持 8BITMIME 扩展，
	// 允许直接传输 UTF-8 字符，无需 Base64 或 Quoted-Printable 额外编码。
	writeHeader(&buf, "Content-Transfer-Encoding", "8bit")
	buf.WriteString("\r\n")
	buf.WriteString(body)
	return buf.Bytes()
}

func writeHeader(buf *bytes.Buffer, key, value string) {
	buf.WriteString(key)
	buf.WriteString(": ")
	buf.WriteString(strings.ReplaceAll(value, "\n", " "))
	buf.WriteString("\r\n")
}

func writePart(buf *bytes.Buffer, boundary, contentType, body string) {
	buf.WriteString("--" + boundary + "\r\n")
	writeHeader(buf, "Content-Type", contentType)
	writeHeader(buf, "Content-Transfer-Encoding", "8bit")
	buf.WriteString("\r\n")
	buf.WriteString(body)
	buf.WriteString("\r\n")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
