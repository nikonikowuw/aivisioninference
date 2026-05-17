package task

import (
	"context"
	"encoding/json"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"
)

// Task type constants.
const (
	TypeEmailDelivery = "email:delivery"
	TypeDataExport    = "data:export"
)

// RegisterHandlers registers task handlers with the mux.
func RegisterHandlers(mux *asynq.ServeMux) {
	mux.HandleFunc(TypeEmailDelivery, handleEmailDelivery)
	mux.HandleFunc(TypeDataExport, handleDataExport)
}

func handleEmailDelivery(ctx context.Context, t *asynq.Task) error {
	var payload struct {
		To      string `json:"to"`
		Subject string `json:"subject"`
		Body    string `json:"body"`
	}
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return err
	}
	zap.L().Info("sending email",
		zap.String("to", payload.To),
		zap.String("subject", payload.Subject),
	)
	// TODO: implement actual email sending
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
