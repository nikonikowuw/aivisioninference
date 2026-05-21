package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/model"
	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
	"github.com/niko-admin/niko-admin/internal/pkg/hash"
	mailpkg "github.com/niko-admin/niko-admin/internal/pkg/mail"
	"github.com/niko-admin/niko-admin/internal/repository"
)

const (
	emailTokenTTL       = 30 * time.Minute
	emailSendWindow     = 10 * time.Minute
	emailSendWindowMax  = 3
	defaultSMTPTimeout  = 10
	defaultIMAPSyncMins = 10
)

// mailConfigRepo 邮件配置持久化接口
type mailConfigRepo interface {
	First(ctx context.Context) (*model.MailConfig, error)
	Save(ctx context.Context, cfg *model.MailConfig) error
}

// inboundEmailRepo 收件箱邮件持久化接口
type inboundEmailRepo interface {
	Exists(ctx context.Context, account, mailbox string, uid uint32, messageID string) (bool, error)
	Create(ctx context.Context, item *model.InboundEmail) error
}

// feedbackRepo 用户反馈持久化接口
type feedbackRepo interface {
	Create(ctx context.Context, item *model.Feedback) error
}

// smtpClient SMTP 发件客户端接口
type smtpClient interface {
	Send(ctx context.Context, msg mailpkg.Message) error
}

// imapClient IMAP 收件客户端接口
type imapClient interface {
	TestConnection(ctx context.Context) error
	FetchRecent(ctx context.Context, limit int) ([]mailpkg.InboundMessage, error)
}

// emailTokenRepo 邮件验证令牌持久化接口
type emailTokenRepo interface {
	Create(ctx context.Context, item *model.EmailToken) error
	FindByHash(ctx context.Context, tokenHash, purpose string) (*model.EmailToken, error)
	CountSince(ctx context.Context, email, purpose, requestIP string, since time.Time) (int64, error)
	Update(ctx context.Context, item *model.EmailToken) error
}

// emailUserRepo 用户邮箱状态更新及密码重置接口
type emailUserRepo interface {
	FindByEmail(ctx context.Context, email string) (*model.User, error)
	MarkEmailVerified(ctx context.Context, userID, email string) error
	UpdatePassword(ctx context.Context, userID, hashedPassword string) error
}

// verificationMailSender 验证邮件发送器接口
type verificationMailSender interface {
	Send(ctx context.Context, msg mailpkg.Message) error
}

// MailService 处理系统邮件配置与邮件收发协议（SMTP/IMAP）的相关业务逻辑
type MailService struct {
	cfgRepo      mailConfigRepo                          // 邮件配置持久化层
	inboundRepo  inboundEmailRepo                        // 接收邮件持久化层
	feedbackRepo feedbackRepo                           // 系统反馈持久化层
	smtpFactory  func(cfg mailpkg.SMTPConfig) smtpClient // SMTP 客户端工厂函数
	imapFactory  func(cfg mailpkg.IMAPConfig) imapClient // IMAP 客户端工厂函数
}

// NewMailService 创建并返回一个新的 MailService 实例
func NewMailService(cfgRepo *repository.MailConfigRepository, inboundRepo *repository.InboundEmailRepository, feedbackRepo *repository.FeedbackRepository) *MailService {
	return &MailService{
		cfgRepo:      cfgRepo,
		inboundRepo:  inboundRepo,
		feedbackRepo: feedbackRepo,
		smtpFactory: func(cfg mailpkg.SMTPConfig) smtpClient {
			return mailpkg.NewSMTPClient(cfg)
		},
		imapFactory: func(cfg mailpkg.IMAPConfig) imapClient {
			return mailpkg.NewIMAPClient(cfg)
		},
	}
}

// GetConfig 获取当前的邮件服务器配置信息
func (s *MailService) GetConfig(ctx context.Context) (*dto.MailConfigResponse, error) {
	cfg, err := s.getOrDefaultConfig(ctx)
	if err != nil {
		zap.L().Error("get mail config failed", zap.Error(err))
		return nil, apperrors.New(apperrors.ErrInternal, "")
	}
	return toMailConfigResponse(cfg), nil
}

// SaveConfig 保存或更新邮件服务器配置信息
func (s *MailService) SaveConfig(ctx context.Context, req dto.MailConfigRequest) (*dto.MailConfigResponse, error) {
	cfg, err := s.getOrDefaultConfig(ctx)
	if err != nil {
		zap.L().Error("get mail config before save failed", zap.Error(err))
		return nil, apperrors.New(apperrors.ErrInternal, "")
	}

	applyMailConfigRequest(cfg, req)
	if err := s.cfgRepo.Save(ctx, cfg); err != nil {
		zap.L().Error("save mail config failed", zap.Error(err))
		return nil, apperrors.New(apperrors.ErrInternal, "")
	}
	return toMailConfigResponse(cfg), nil
}

// TestSMTP 用指定收件人邮箱测试 SMTP 发送服务是否正常
func (s *MailService) TestSMTP(ctx context.Context, to string) error {
	cfg, err := s.getOrDefaultConfig(ctx)
	if err != nil {
		return apperrors.New(apperrors.ErrInternal, "")
	}
	client := s.smtpFactory(toSMTPConfig(cfg))
	if err := client.Send(ctx, mailpkg.Message{
		To:       []string{to},
		Subject:  "Niko Admin SMTP Test",
		TextBody: "This is a test email from Niko Admin.",
		HTMLBody: "<p>This is a test email from Niko Admin.</p>",
	}); err != nil {
		zap.L().Warn("smtp test failed", zap.String("to", to), zap.Error(sanitizeMailError(err)))
		return apperrors.New(apperrors.ErrSMTPTestFailed, "")
	}
	return nil
}

// TestIMAP 测试 IMAP 收件服务器连接是否正常
func (s *MailService) TestIMAP(ctx context.Context) error {
	cfg, err := s.getOrDefaultConfig(ctx)
	if err != nil {
		return apperrors.New(apperrors.ErrInternal, "")
	}
	client := s.imapFactory(toIMAPConfig(cfg))
	if err := client.TestConnection(ctx); err != nil {
		zap.L().Warn("imap test failed", zap.Error(sanitizeMailError(err)))
		return apperrors.New(apperrors.ErrIMAPTestFailed, "")
	}
	return nil
}

// Send 调用 SMTP 发送一封邮件
func (s *MailService) Send(ctx context.Context, msg mailpkg.Message) error {
	cfg, err := s.getOrDefaultConfig(ctx)
	if err != nil {
		return err
	}
	return s.smtpFactory(toSMTPConfig(cfg)).Send(ctx, msg)
}

// NotificationAddress 获取接收系统通知的回复/管理员邮箱地址
func (s *MailService) NotificationAddress(ctx context.Context) (string, error) {
	cfg, err := s.getOrDefaultConfig(ctx)
	if err != nil {
		return "", err
	}
	return cfg.ReplyTo, nil
}

// SyncIMAP 从 IMAP 接收新邮件并自动同步转换为系统反馈记录
func (s *MailService) SyncIMAP(ctx context.Context, limit int) (int, error) {
	cfg, err := s.getOrDefaultConfig(ctx)
	if err != nil {
		return 0, err
	}
	if !cfg.IMAPEnabled {
		return 0, nil
	}
	messages, err := s.imapFactory(toIMAPConfig(cfg)).FetchRecent(ctx, limit)
	if err != nil {
		return 0, err
	}

	saved := 0
	for _, message := range messages {
		// 以 UID + MessageID 作为去重依据，同一封邮件被多次同步时不会重复插入。
		exists, err := s.inboundRepo.Exists(ctx, cfg.IMAPUsername, cfg.IMAPMailbox, message.UID, message.MessageID)
		if err != nil {
			return saved, err
		}
		if exists {
			continue
		}
		// 同步 IMAP 收到的邮件映射为 InboundEmail + Feedback 两条记录，
		// 实现「客户发邮件→自动创建反馈」的业务流程。
		inbound := &model.InboundEmail{
			Account:   cfg.IMAPUsername,
			Mailbox:   cfg.IMAPMailbox,
			UID:       message.UID,
			MessageID: message.MessageID,
			From:      message.From,
			To:        message.To,
			Subject:   message.Subject,
			TextBody:  message.TextBody,
			HTMLBody:  message.HTMLBody,
			Summary:   message.Summary,
			EmailDate: message.Date,
			Status:    model.InboundEmailStatusNew,
			RawSize:   message.RawSize,
		}
		if err := s.inboundRepo.Create(ctx, inbound); err != nil {
			return saved, err
		}
		title := message.Subject
		if title == "" {
			title = "Email feedback"
		}
		content := message.TextBody
		if content == "" {
			content = message.Summary
		}
		// 将邮件映射为 Feedback 记录，InboundEmailID 关联原始邮件以便追溯。
		feedback := &model.Feedback{
			Source:         model.FeedbackSourceEmail,
			Category:       "email",
			Title:          title,
			Content:        content,
			Email:          message.From,
			InboundEmailID: &inbound.ID,
			Status:         model.FeedbackStatusOpen,
		}
		if err := s.feedbackRepo.Create(ctx, feedback); err != nil {
			return saved, err
		}
		saved++
	}
	return saved, nil
}

// getOrDefaultConfig 获取邮件配置，若不存在则返回带默认值的占位配置
func (s *MailService) getOrDefaultConfig(ctx context.Context) (*model.MailConfig, error) {
	cfg, err := s.cfgRepo.First(ctx)
	if err != nil {
		return nil, err
	}
	if cfg != nil {
		return cfg, nil
	}
	return &model.MailConfig{
		SMTPEncryption:  model.MailEncryptionSTARTTLS,
		SMTPTimeoutSec:  defaultSMTPTimeout,
		IMAPEncryption:  model.MailEncryptionTLS,
		IMAPMailbox:     "INBOX",
		IMAPSyncMinutes: defaultIMAPSyncMins,
	}, nil
}

// EmailVerificationService 处理一次性电子邮件验证及密码重置令牌的业务逻辑
type EmailVerificationService struct {
	tokenRepo emailTokenRepo         // 验证令牌持久化层
	userRepo  emailUserRepo          // 用户持久化层
	mailSvc   verificationMailSender // 验证邮件发送服务
}

// NewEmailVerificationService 创建并返回一个新的 EmailVerificationService 实例
func NewEmailVerificationService(tokenRepo *repository.EmailTokenRepository, userRepo *repository.UserRepository, mailSvc *MailService) *EmailVerificationService {
	return &EmailVerificationService{tokenRepo: tokenRepo, userRepo: userRepo, mailSvc: mailSvc}
}

// SendVerification 创建并发送一封邮箱所有权验证邮件
func (s *EmailVerificationService) SendVerification(ctx context.Context, userID, email, requestIP string) error {
	return s.createAndSend(ctx, &userID, email, model.EmailTokenPurposeChangeVerify, requestIP, "Verify your email")
}

// SendPasswordReset 创建并发送一封重置密码用的验证邮件
func (s *EmailVerificationService) SendPasswordReset(ctx context.Context, email, requestIP string) error {
	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		return nil
	}
	return s.createAndSend(ctx, &user.ID, email, model.EmailTokenPurposePasswordReset, requestIP, "Reset your password")
}

// VerifyToken 验证指定用途的令牌，验证成功后将令牌消费并标记邮箱为已验证
func (s *EmailVerificationService) VerifyToken(ctx context.Context, token, purpose string) error {
	item, err := s.consumeToken(ctx, token, purpose)
	if err != nil {
		return err
	}
	if err := s.tokenRepo.Update(ctx, item); err != nil {
		return apperrors.New(apperrors.ErrInternal, "")
	}
	if purpose == model.EmailTokenPurposeChangeVerify && item.UserID != nil {
		if err := s.userRepo.MarkEmailVerified(ctx, *item.UserID, item.Email); err != nil {
			return apperrors.New(apperrors.ErrInternal, "")
		}
	}
	return nil
}

// ResetPassword 使用密码重置令牌对用户进行密码重置
func (s *EmailVerificationService) ResetPassword(ctx context.Context, token, newPassword string) error {
	item, err := s.consumeToken(ctx, token, model.EmailTokenPurposePasswordReset)
	if err != nil {
		return err
	}
	if item.UserID == nil {
		return apperrors.New(apperrors.ErrTokenInvalidOrExpired, "")
	}
	hashedPassword, err := hashPassword(newPassword)
	if err != nil {
		return apperrors.New(apperrors.ErrInternal, "")
	}
	if err := s.userRepo.UpdatePassword(ctx, *item.UserID, hashedPassword); err != nil {
		return apperrors.New(apperrors.ErrInternal, "")
	}
	if err := s.tokenRepo.Update(ctx, item); err != nil {
		return apperrors.New(apperrors.ErrInternal, "")
	}
	return nil
}

// consumeToken 消费一个一次性令牌，检验其哈希、用途及过期状态，防范重放攻击
func (s *EmailVerificationService) consumeToken(ctx context.Context, token, purpose string) (*model.EmailToken, error) {
	tokenHash := hashToken(token)
	item, err := s.tokenRepo.FindByHash(ctx, tokenHash, purpose)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrTokenInvalidOrExpired, "")
	}
	// UsedAt != nil 表示该 token 已被消费，防止重放攻击：拦截二次使用同一 token 的请求。
	// 过期时间检查确保即使 token 未被使用，超过 TTL 后也无法再使用。
	if item.UsedAt != nil || time.Now().After(item.ExpiresAt) {
		return nil, apperrors.New(apperrors.ErrTokenInvalidOrExpired, "")
	}
	now := time.Now()
	item.UsedAt = &now
	return item, nil
}

// createAndSend 生成一封包含一次性验证令牌的邮件并发送，且带有限流防护
func (s *EmailVerificationService) createAndSend(ctx context.Context, userID *string, email, purpose, requestIP, subject string) error {
	// 限流：同一邮箱在同一时间窗口内的发信次数不能超过上限，防止暴力调用。
	count, err := s.tokenRepo.CountSince(ctx, email, purpose, requestIP, time.Now().Add(-emailSendWindow))
	if err != nil {
		return apperrors.New(apperrors.ErrInternal, "")
	}
	if count >= emailSendWindowMax {
		return apperrors.New(apperrors.ErrBadRequest, "")
	}

	token, err := randomToken()
	if err != nil {
		return apperrors.New(apperrors.ErrInternal, "")
	}
	item := &model.EmailToken{
		UserID:    userID,
		Email:     email,
		Purpose:   purpose,
		TokenHash: hashToken(token),
		ExpiresAt: time.Now().Add(emailTokenTTL),
		RequestIP: requestIP,
	}
	if err := s.tokenRepo.Create(ctx, item); err != nil {
		return apperrors.New(apperrors.ErrInternal, "")
	}
	// Best-effort 邮件发送：邮件发送失败时记录日志，但令牌已落库，数据库状态允许重试。
	body := fmt.Sprintf("Your verification code is: %s\nThis code expires in 30 minutes.", token)
	if err := s.mailSvc.Send(ctx, mailpkg.Message{To: []string{email}, Subject: subject, TextBody: body}); err != nil {
		zap.L().Warn("send verification email failed", zap.String("email", email), zap.Error(sanitizeMailError(err)))
		return apperrors.New(apperrors.ErrSMTPTestFailed, "")
	}
	return nil
}

func toMailConfigResponse(cfg *model.MailConfig) *dto.MailConfigResponse {
	return &dto.MailConfigResponse{
		ID:                     cfg.ID,
		Enabled:                cfg.Enabled,
		FromName:               cfg.FromName,
		FromAddress:            cfg.FromAddress,
		ReplyTo:                cfg.ReplyTo,
		SMTPEnabled:            cfg.SMTPEnabled,
		SMTPHost:               cfg.SMTPHost,
		SMTPPort:               cfg.SMTPPort,
		SMTPUsername:           cfg.SMTPUsername,
		SMTPPasswordConfigured: cfg.SMTPPassword != "",
		SMTPEncryption:         cfg.SMTPEncryption,
		SMTPTimeoutSec:         cfg.SMTPTimeoutSec,
		IMAPEnabled:            cfg.IMAPEnabled,
		IMAPHost:               cfg.IMAPHost,
		IMAPPort:               cfg.IMAPPort,
		IMAPUsername:           cfg.IMAPUsername,
		IMAPPasswordConfigured: cfg.IMAPPassword != "",
		IMAPEncryption:         cfg.IMAPEncryption,
		IMAPMailbox:            cfg.IMAPMailbox,
		IMAPSyncMinutes:        cfg.IMAPSyncMinutes,
		CreatedAt:              cfg.CreatedAt.Format(dto.DateTimeFormat),
		UpdatedAt:              cfg.UpdatedAt.Format(dto.DateTimeFormat),
	}
}

func applyMailConfigRequest(cfg *model.MailConfig, req dto.MailConfigRequest) {
	cfg.Enabled = req.Enabled
	cfg.FromName = strings.TrimSpace(req.FromName)
	cfg.FromAddress = strings.TrimSpace(req.FromAddress)
	cfg.ReplyTo = strings.TrimSpace(req.ReplyTo)
	cfg.SMTPEnabled = req.SMTPEnabled
	cfg.SMTPHost = strings.TrimSpace(req.SMTPHost)
	cfg.SMTPPort = req.SMTPPort
	cfg.SMTPUsername = strings.TrimSpace(req.SMTPUsername)
	if req.SMTPPassword != "" {
		cfg.SMTPPassword = req.SMTPPassword
	}
	cfg.SMTPEncryption = defaultString(req.SMTPEncryption, model.MailEncryptionSTARTTLS)
	cfg.SMTPTimeoutSec = defaultInt(req.SMTPTimeoutSec, defaultSMTPTimeout)
	cfg.IMAPEnabled = req.IMAPEnabled
	cfg.IMAPHost = strings.TrimSpace(req.IMAPHost)
	cfg.IMAPPort = req.IMAPPort
	cfg.IMAPUsername = strings.TrimSpace(req.IMAPUsername)
	if req.IMAPPassword != "" {
		cfg.IMAPPassword = req.IMAPPassword
	}
	cfg.IMAPEncryption = defaultString(req.IMAPEncryption, model.MailEncryptionTLS)
	cfg.IMAPMailbox = defaultString(req.IMAPMailbox, "INBOX")
	cfg.IMAPSyncMinutes = defaultInt(req.IMAPSyncMinutes, defaultIMAPSyncMins)
}

func toSMTPConfig(cfg *model.MailConfig) mailpkg.SMTPConfig {
	return mailpkg.SMTPConfig{
		Enabled:     cfg.Enabled && cfg.SMTPEnabled,
		Host:        cfg.SMTPHost,
		Port:        cfg.SMTPPort,
		Username:    cfg.SMTPUsername,
		Password:    cfg.SMTPPassword,
		Encryption:  cfg.SMTPEncryption,
		Timeout:     time.Duration(defaultInt(cfg.SMTPTimeoutSec, defaultSMTPTimeout)) * time.Second,
		FromName:    cfg.FromName,
		FromAddress: cfg.FromAddress,
		ReplyTo:     cfg.ReplyTo,
	}
}

func toIMAPConfig(cfg *model.MailConfig) mailpkg.IMAPConfig {
	return mailpkg.IMAPConfig{
		Enabled:    cfg.Enabled && cfg.IMAPEnabled,
		Host:       cfg.IMAPHost,
		Port:       cfg.IMAPPort,
		Username:   cfg.IMAPUsername,
		Password:   cfg.IMAPPassword,
		Encryption: cfg.IMAPEncryption,
		Mailbox:    cfg.IMAPMailbox,
		Timeout:    time.Duration(defaultInt(cfg.SMTPTimeoutSec, defaultSMTPTimeout)) * time.Second,
	}
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}

func defaultInt(value, fallback int) int {
	if value <= 0 {
		return fallback
	}
	return value
}

func randomToken() (string, error) {
	var b [18]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b[:]), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func hashPassword(password string) (string, error) {
	return hash.Hash(password)
}

func sanitizeMailError(err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	for _, marker := range []string{"password", "AUTH", "LOGIN"} {
		msg = strings.ReplaceAll(msg, marker, "[redacted]")
	}
	return fmt.Errorf("%s", msg)
}
