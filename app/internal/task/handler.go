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

// Task type constants.
const (
	TypeEmailDelivery = "email:delivery"
	TypeEmailSync     = "email:sync"
	TypeDataExport    = "data:export"
)

// Handler holds dependencies for background task processing.
type Handler struct {
	mailSvc *service.MailService
}

// NewHandler creates a new Handler with the given dependencies.
func NewHandler(mailSvc *service.MailService) *Handler {
	return &Handler{mailSvc: mailSvc}
}

// RegisterHandlers registers task handlers with the mux.
func (h *Handler) RegisterHandlers(mux *asynq.ServeMux) {
	mux.HandleFunc(TypeEmailDelivery, h.handleEmailDelivery)
	mux.HandleFunc(TypeEmailSync, h.handleEmailSync)
	mux.HandleFunc(TypeDataExport, handleDataExport)
}

func (h *Handler) handleEmailDelivery(ctx context.Context, t *asynq.Task) error {
	var payload struct {
		To       string   `json:"to"`
		ToList   []string `json:"to_list"`
		Subject  string   `json:"subject"`
		Body     string   `json:"body"`
		TextBody string   `json:"text_body"`
		HTMLBody string   `json:"html_body"`
	}
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return err
	}
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
	return h.mailSvc.Send(ctx, mailpkg.Message{
		To:       recipients,
		Subject:  payload.Subject,
		TextBody: body,
		HTMLBody: payload.HTMLBody,
	})
}

func (h *Handler) handleEmailSync(ctx context.Context, t *asynq.Task) error {
	var payload struct {
		Limit int `json:"limit"`
	}
	if len(t.Payload()) > 0 {
		if err := json.Unmarshal(t.Payload(), &payload); err != nil {
			return err
		}
	}
	count, err := h.mailSvc.SyncIMAP(ctx, payload.Limit)
	if err != nil {
		return err
	}
	zap.L().Info("imap sync completed", zap.Int("synced", count))
	return nil
}

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
	// TODO: implement actual data export
	return nil
}
