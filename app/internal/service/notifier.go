package service

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/smtp"
	"strings"
	"sync"
	"time"

	"github.com/niko-admin/niko-admin/internal/model"
	"go.uber.org/zap"
)

// NotifierConfig holds configuration for all notification providers.
type NotifierConfig struct {
	// Webhook settings
	WebhookURL string `json:"webhook_url" yaml:"webhook_url"`

	// SMTP/Email settings
	SMTPServer   string `json:"smtp_server" yaml:"smtp_server"`
	SMTPPort     int    `json:"smtp_port" yaml:"smtp_port"`
	SMTPUsername string `json:"smtp_username" yaml:"smtp_username"`
	SMTPPassword string `json:"smtp_password" yaml:"smtp_password"`
	SMTPFrom     string `json:"smtp_from" yaml:"smtp_from"`
	SMTPUseTLS   bool   `json:"smtp_use_tls" yaml:"smtp_use_tls"`
	SMTPTo       string `json:"smtp_to" yaml:"smtp_to"` // default recipient

	// Telegram settings
	TelegramBotToken string `json:"telegram_bot_token" yaml:"telegram_bot_token"`
	TelegramChatID   string `json:"telegram_chat_id" yaml:"telegram_chat_id"`

	// DingTalk settings
	DingTalkWebhookURL string `json:"dingtalk_webhook_url" yaml:"dingtalk_webhook_url"`

	// Feishu/Lark settings
	FeishuWebhookURL string `json:"feishu_webhook_url" yaml:"feishu_webhook_url"`

	// WeCom/WeChat Work settings
	WeComWebhookURL string `json:"wecom_webhook_url" yaml:"wecom_webhook_url"`
}

// AlertNotificationPayload 告警通知负载
type AlertNotificationPayload struct {
	Title       string            `json:"title"`
	Message     string            `json:"message"`
	EventID     string            `json:"event_id"`
	RuleName    string            `json:"rule_name"`
	RuleID      string            `json:"rule_id"`
	NodeID      string            `json:"node_id"`
	NodeName    string            `json:"node_name,omitempty"`
	MetricType  string            `json:"metric_type"`
	MetricValue float64           `json:"metric_value"`
	Threshold   float64           `json:"threshold"`
	Operator    string            `json:"operator"`
	Status      string            `json:"status"`
	FiredAt     string            `json:"fired_at"`
	Extra       map[string]string `json:"extra,omitempty"`
}

// Notifier defines the interface for notification providers.
type Notifier interface {
	// Name returns the channel name for this notifier.
	Name() string
	// Send sends a notification for the given alert event.
	Send(ctx context.Context, event *model.AlertEvent, rule *model.AlertRule, nodeName string) error
}

// NotifierRegistry manages available notification providers.
type NotifierRegistry struct {
	mu        sync.RWMutex
	providers map[string]Notifier
}

// NewNotifierRegistry creates a new NotifierRegistry.
func NewNotifierRegistry() *NotifierRegistry {
	return &NotifierRegistry{
		providers: make(map[string]Notifier),
	}
}

// Register adds a notifier provider.
func (r *NotifierRegistry) Register(n Notifier) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[n.Name()] = n
}

// Get returns a notifier by channel name.
func (r *NotifierRegistry) Get(name string) (Notifier, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	n, ok := r.providers[name]
	return n, ok
}

// GetActive returns notifiers for the given channel names.
func (r *NotifierRegistry) GetActive(channels []string) []Notifier {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var active []Notifier
	for _, ch := range channels {
		if n, ok := r.providers[ch]; ok {
			active = append(active, n)
		}
	}
	return active
}

// SendToChannels sends a notification through the specified channels.
func (r *NotifierRegistry) SendToChannels(ctx context.Context, channels []string, event *model.AlertEvent, rule *model.AlertRule, nodeName string) {
	notifiers := r.GetActive(channels)
	if len(notifiers) == 0 {
		zap.L().Warn("no active notifier found for channels",
			zap.Strings("channels", channels),
			zap.String("event_id", event.ID),
		)
		return
	}

	for _, n := range notifiers {
		if err := n.Send(ctx, event, rule, nodeName); err != nil {
			zap.L().Error("notification send failed",
				zap.String("provider", n.Name()),
				zap.String("event_id", event.ID),
				zap.Error(err),
			)
		} else {
			zap.L().Info("notification sent",
				zap.String("provider", n.Name()),
				zap.String("event_id", event.ID),
			)
		}
	}
}

// InitDefaultNotifiers initializes the default set of notifiers from config.
func InitDefaultNotifiers(cfg *NotifierConfig) *NotifierRegistry {
	registry := NewNotifierRegistry()

	if cfg.WebhookURL != "" {
		registry.Register(NewWebhookNotifier(cfg.WebhookURL))
		zap.L().Info("registered webhook notifier")
	}

	if cfg.SMTPServer != "" && cfg.SMTPFrom != "" {
		registry.Register(NewEmailNotifier(cfg))
		zap.L().Info("registered email notifier")
	}

	if cfg.TelegramBotToken != "" && cfg.TelegramChatID != "" {
		registry.Register(NewTelegramNotifier(cfg.TelegramBotToken, cfg.TelegramChatID))
		zap.L().Info("registered telegram notifier")
	}

	if cfg.DingTalkWebhookURL != "" {
		registry.Register(NewDingTalkNotifier(cfg.DingTalkWebhookURL))
		zap.L().Info("registered dingtalk notifier")
	}

	if cfg.FeishuWebhookURL != "" {
		registry.Register(NewFeishuNotifier(cfg.FeishuWebhookURL))
		zap.L().Info("registered feishu notifier")
	}

	if cfg.WeComWebhookURL != "" {
		registry.Register(NewWeComNotifier(cfg.WeComWebhookURL))
		zap.L().Info("registered wecom notifier")
	}

	return registry
}

// buildAlertPayload creates a common notification payload from alert event + rule.
func buildAlertPayload(event *model.AlertEvent, rule *model.AlertRule, nodeName string) AlertNotificationPayload {
	title := fmt.Sprintf("[%s] %s", strings.ToUpper(event.Status), rule.Name)
	message := fmt.Sprintf("规则: %s\n节点: %s\n指标: %s %.1f %s %.1f\n触发时间: %s",
		rule.Name,
		nodeName,
		rule.MetricType,
		event.MetricValue,
		rule.Operator,
		rule.Threshold,
		event.FiredAt.Format(time.RFC3339),
	)

	return AlertNotificationPayload{
		Title:       title,
		Message:     message,
		EventID:     event.ID,
		RuleName:    rule.Name,
		RuleID:      rule.ID,
		NodeID:      event.NodeID,
		NodeName:    nodeName,
		MetricType:  rule.MetricType,
		MetricValue: event.MetricValue,
		Threshold:   rule.Threshold,
		Operator:    rule.Operator,
		Status:      event.Status,
		FiredAt:     event.FiredAt.Format(time.RFC3339),
	}
}

// ---------------------------------------------------------------------------
// Webhook Notifier
// ---------------------------------------------------------------------------

// WebhookNotifier sends alert notifications via HTTP POST to a configurable URL.
type WebhookNotifier struct {
	url string
}

// NewWebhookNotifier creates a new WebhookNotifier.
func NewWebhookNotifier(url string) *WebhookNotifier {
	return &WebhookNotifier{url: url}
}

func (n *WebhookNotifier) Name() string {
	return "webhook"
}

func (n *WebhookNotifier) Send(ctx context.Context, event *model.AlertEvent, rule *model.AlertRule, nodeName string) error {
	payload := buildAlertPayload(event, rule, nodeName)

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("webhook: marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("webhook: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "AIVisionInference-Alert/1.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("webhook: send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("webhook: unexpected status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// ---------------------------------------------------------------------------
// Email Notifier (SMTP)
// ---------------------------------------------------------------------------

// EmailNotifier sends alert notifications via SMTP email.
type EmailNotifier struct {
	config *NotifierConfig
}

// NewEmailNotifier creates a new EmailNotifier.
func NewEmailNotifier(cfg *NotifierConfig) *EmailNotifier {
	return &EmailNotifier{config: cfg}
}

func (n *EmailNotifier) Name() string {
	return "email"
}

func (n *EmailNotifier) Send(ctx context.Context, event *model.AlertEvent, rule *model.AlertRule, nodeName string) error {
	payload := buildAlertPayload(event, rule, nodeName)

	to := n.config.SMTPTo
	if to == "" {
		return fmt.Errorf("email: no recipient configured (smtp_to)")
	}

	subject := fmt.Sprintf("=?UTF-8?B?%s?=",
		base64.StdEncoding.EncodeToString([]byte(payload.Title)))
	body := payload.Message

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		n.config.SMTPFrom, to, subject, body)

	addr := fmt.Sprintf("%s:%d", n.config.SMTPServer, n.config.SMTPPort)
	auth := smtp.PlainAuth("", n.config.SMTPUsername, n.config.SMTPPassword, n.config.SMTPServer)

	if n.config.SMTPUseTLS {
		return sendTLSMail(addr, auth, n.config.SMTPFrom, []string{to}, []byte(msg))
	}

	return smtp.SendMail(addr, auth, n.config.SMTPFrom, []string{to}, []byte(msg))
}

func sendTLSMail(addr string, auth smtp.Auth, from string, to []string, msg []byte) error {
	tlsConfig := &tls.Config{InsecureSkipVerify: false}
	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("tls dial: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, addr)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	defer client.Close()

	if auth != nil {
		if err = client.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}

	if err = client.Mail(from); err != nil {
		return fmt.Errorf("smtp mail from: %w", err)
	}

	for _, addr := range to {
		if err = client.Rcpt(addr); err != nil {
			return fmt.Errorf("smtp rcpt %s: %w", addr, err)
		}
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	defer w.Close()

	if _, err = w.Write(msg); err != nil {
		return fmt.Errorf("smtp write: %w", err)
	}

	return client.Quit()
}

// ---------------------------------------------------------------------------
// Telegram Notifier
// ---------------------------------------------------------------------------

// TelegramNotifier sends notifications via Telegram Bot API.
type TelegramNotifier struct {
	botToken string
	chatID   string
}

// NewTelegramNotifier creates a new TelegramNotifier.
func NewTelegramNotifier(botToken, chatID string) *TelegramNotifier {
	return &TelegramNotifier{botToken: botToken, chatID: chatID}
}

func (n *TelegramNotifier) Name() string {
	return "telegram"
}

func (n *TelegramNotifier) Send(ctx context.Context, event *model.AlertEvent, rule *model.AlertRule, nodeName string) error {
	payload := buildAlertPayload(event, rule, nodeName)

	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", n.botToken)

	body := map[string]interface{}{
		"chat_id":    n.chatID,
		"text":       payload.Message,
		"parse_mode": "Markdown",
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("telegram: marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("telegram: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("telegram: send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telegram: unexpected status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// ---------------------------------------------------------------------------
// DingTalk Notifier
// ---------------------------------------------------------------------------

// DingTalkNotifier sends notifications via DingTalk robot webhook.
type DingTalkNotifier struct {
	webhookURL string
}

// NewDingTalkNotifier creates a new DingTalkNotifier.
func NewDingTalkNotifier(webhookURL string) *DingTalkNotifier {
	return &DingTalkNotifier{webhookURL: webhookURL}
}

func (n *DingTalkNotifier) Name() string {
	return "dingtalk"
}

func (n *DingTalkNotifier) Send(ctx context.Context, event *model.AlertEvent, rule *model.AlertRule, nodeName string) error {
	payload := buildAlertPayload(event, rule, nodeName)

	body := map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]string{
			"title": payload.Title,
			"text":  fmt.Sprintf("### %s\n\n%s", payload.Title, payload.Message),
		},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("dingtalk: marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.webhookURL, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("dingtalk: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("dingtalk: send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("dingtalk: unexpected status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// ---------------------------------------------------------------------------
// Feishu Notifier
// ---------------------------------------------------------------------------

// FeishuNotifier sends notifications via Feishu/Lark robot webhook.
type FeishuNotifier struct {
	webhookURL string
}

// NewFeishuNotifier creates a new FeishuNotifier.
func NewFeishuNotifier(webhookURL string) *FeishuNotifier {
	return &FeishuNotifier{webhookURL: webhookURL}
}

func (n *FeishuNotifier) Name() string {
	return "feishu"
}

func (n *FeishuNotifier) Send(ctx context.Context, event *model.AlertEvent, rule *model.AlertRule, nodeName string) error {
	payload := buildAlertPayload(event, rule, nodeName)

	body := map[string]interface{}{
		"msg_type": "interactive",
		"card": map[string]interface{}{
			"header": map[string]interface{}{
				"title": map[string]string{
					"tag":  "plain_text",
					"content": payload.Title,
				},
			},
			"elements": []map[string]interface{}{
				{
					"tag":  "markdown",
					"content": fmt.Sprintf("**规则**: %s\n**节点**: %s\n**指标**: %s\n**当前值**: %.1f\n**阈值**: %s %.1f\n**触发时间**: %s",
						rule.Name, nodeName, rule.MetricType,
						event.MetricValue, rule.Operator, rule.Threshold,
						event.FiredAt.Format(time.RFC3339),
					),
				},
			},
		},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("feishu: marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.webhookURL, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("feishu: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("feishu: send request: %w", err)
	}
	defer resp.Body.Close()

	// Feishu returns 200 even for errors - read the body
	respBody, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	if json.Unmarshal(respBody, &result) == nil {
		if code, ok := result["code"].(float64); ok && code != 0 {
			return fmt.Errorf("feishu: api error code %.0f: %v", code, result["msg"])
		}
	}

	return nil
}

// ---------------------------------------------------------------------------
// WeCom (WeChat Work) Notifier
// ---------------------------------------------------------------------------

// WeComNotifier sends notifications via WeCom robot webhook.
type WeComNotifier struct {
	webhookURL string
}

// NewWeComNotifier creates a new WeComNotifier.
func NewWeComNotifier(webhookURL string) *WeComNotifier {
	return &WeComNotifier{webhookURL: webhookURL}
}

func (n *WeComNotifier) Name() string {
	return "wecom"
}

func (n *WeComNotifier) Send(ctx context.Context, event *model.AlertEvent, rule *model.AlertRule, nodeName string) error {
	payload := buildAlertPayload(event, rule, nodeName)

	body := map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]string{
			"content": fmt.Sprintf("## %s\n%s", payload.Title, payload.Message),
		},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("wecom: marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.webhookURL, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("wecom: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("wecom: send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("wecom: unexpected status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}
