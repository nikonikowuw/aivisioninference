package task

import (
	"context"
	"encoding/json"
	"log"

	"github.com/hibiken/asynq"
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
	log.Printf("Sending email to %s: %s", payload.To, payload.Subject)
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
	log.Printf("Exporting data for user %s in %s format", payload.UserID, payload.Format)
	// TODO: implement actual data export
	return nil
}
