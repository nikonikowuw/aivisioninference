// Package task 提供基于 Asynq 的后台异步任务队列管理功能
package task

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"

	mailpkg "github.com/niko-admin/niko-admin/internal/pkg/mail"
	"github.com/niko-admin/niko-admin/internal/service"
)

// 后台异步任务类型常量定义
const (
	TypeEmailDelivery = "email:delivery" // 邮件投递任务
	TypeEmailSync     = "email:sync"     // IMAP 邮件同步任务
	TypeDataExport    = "data:export"    // 数据导出任务
)

// Handler 包含了后台异步任务处理所需的依赖项
type Handler struct {
	mailSvc           *service.MailService        // 邮件服务依赖
	deviceStatusHandler *DeviceStatusHandler // 设备状态检查处理器
}

// NewHandler 创建并返回一个任务处理 Handler 实例
func NewHandler(mailSvc *service.MailService, deviceStatusHandler *DeviceStatusHandler) *Handler {
	return &Handler{
		mailSvc:           mailSvc,
		deviceStatusHandler: deviceStatusHandler,
	}
}

// RegisterHandlers 将所有具体的任务处理函数注册到 Asynq 的路由多路复用器 (Mux) 中
func (h *Handler) RegisterHandlers(mux *asynq.ServeMux) {
	mux.HandleFunc(TypeEmailDelivery, h.handleEmailDelivery)
	mux.HandleFunc(TypeEmailSync, h.handleEmailSync)
	mux.HandleFunc(TypeDataExport, handleDataExport)

	// 注册设备状态检查处理器
	if h.deviceStatusHandler != nil {
		h.deviceStatusHandler.RegisterHandlers(mux)
	}
}

// handleEmailDelivery 处理邮件发送任务，解析载荷并调用邮件服务发送邮件
func (h *Handler) handleEmailDelivery(ctx context.Context, t *asynq.Task) error {
	var payload struct {
		To       string   `json:"to"`
		ToList   []string `json:"to_list"`
		Subject  string   `json:"subject"`
		Body     string   `json:"body"`
		TextBody string   `json:"text_body"`
		HTMLBody string   `json:"html_body"`
	}
	// 反序列化任务载荷
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return err
	}

	// 整合收件人列表
	recipients := payload.ToList
	if payload.To != "" {
		recipients = append(recipients, payload.To)
	}

	zap.L().Info("sending email",
		zap.String("to", strings.Join(recipients, ",")),
		zap.String("subject", payload.Subject),
	)

	body := payload.TextBody
	if body == "" {
		body = payload.Body
	}

	// 调用邮件服务发送邮件
	return h.mailSvc.Send(ctx, mailpkg.Message{
		To:       recipients,
		Subject:  payload.Subject,
		TextBody: body,
		HTMLBody: payload.HTMLBody,
	})
}

// handleEmailSync 处理 IMAP 邮件同步任务，同步指定限制数量的收件箱邮件
func (h *Handler) handleEmailSync(ctx context.Context, t *asynq.Task) error {
	var payload struct {
		Limit int `json:"limit"`
	}
	if len(t.Payload()) > 0 {
		if err := json.Unmarshal(t.Payload(), &payload); err != nil {
			return err
		}
	}

	// 调用邮件服务同步 IMAP
	count, err := h.mailSvc.SyncIMAP(ctx, payload.Limit)
	if err != nil {
		return err
	}
	zap.L().Info("imap sync completed", zap.Int("synced", count))
	return nil
}

// handleDataExport 处理异步数据导出任务
func handleDataExport(ctx context.Context, t *asynq.Task) error {
	var payload struct {
		UserID string `json:"user_id"`
		Format string `json:"format"`
	}
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return err
	}
	zap.L().Info("exporting data",
		zap.String("user_id", payload.UserID),
		zap.String("format", payload.Format),
	)
	// TODO: 待实现实际的数据导出逻辑
	return nil
}
