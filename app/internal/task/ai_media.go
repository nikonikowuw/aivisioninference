package task

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"github.com/niko-admin/niko-admin/internal/model"
	"github.com/niko-admin/niko-admin/internal/pkg/zlm"
	"github.com/niko-admin/niko-admin/internal/repository"
	"go.uber.org/zap"
	"gorm.io/datatypes"
)

const (
	// TypeAIScreenshot is the Asynq task type for AI analysis
	TypeAIScreenshot = "ai:screenshot"

	// TypeEventScreenshot is for event-triggered screenshots
	TypeEventScreenshot = "ai:event_screenshot"
)

// AIScreenshotPayload is the payload for AI screenshot tasks.
type AIScreenshotPayload struct {
	DeviceID      string `json:"device_id"`
	SnapshotURL   string `json:"snapshot_url"`
	Algorithms    string `json:"algorithms"` // comma-separated
	Priority      int    `json:"priority"`   // higher = more urgent
	Timestamp     int64  `json:"timestamp"`
	AlertRecordID string `json:"alert_record_id,omitempty"` // associated alert for event-triggered shots
}

// HandleAIScreenshot processes a screenshot frame for AI analysis.
func HandleAIScreenshot(ctx context.Context, t *asynq.Task, zlmClient *zlm.Client, storage *ScreenshotStorage, inferenceRepo *repository.AIInferenceRepository) error {
	var payload AIScreenshotPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return err
	}

	zap.L().Info("processing AI screenshot",
		zap.String("device_id", payload.DeviceID),
		zap.String("algorithms", payload.Algorithms),
		zap.Int("priority", payload.Priority),
	)

	// Capture snapshot from ZLM
	imgData, err := zlmClient.GetSnap(ctx, zlm.GetSnapRequest{
		URL:        payload.SnapshotURL,
		TimeoutSec: 10,
		ExpireSec:  30,
	})
	if err != nil {
		zap.L().Error("failed to capture snapshot for AI", zap.Error(err))
		return fmt.Errorf("zlm snapshot: %w", err)
	}

	// Save screenshot
	filePath, err := storage.Save(payload.DeviceID, imgData, payload.Timestamp)
	if err != nil {
		zap.L().Error("save AI screenshot failed", zap.Error(err))
		return err
	}

	zap.L().Info("AI screenshot saved for analysis",
		zap.String("device_id", payload.DeviceID),
		zap.String("path", filePath),
		zap.Int("image_size", len(imgData)),
	)

	// Save inference result record to database
	result := &model.AIInferenceResult{
		DeviceID:     payload.DeviceID,
		SnapshotPath: filePath,
		Algorithms:   payload.Algorithms,
		Result:       datatypes.JSON([]byte("{}")),
		Confidence:   0.0,
		TaskType:     "event",
		ProcessedAt:  time.Unix(payload.Timestamp, 0),
	}

	if payload.AlertRecordID != "" {
		result.AlertType = "event_triggered"
	}

	if inferenceRepo != nil {
		if err := inferenceRepo.Create(ctx, result); err != nil {
			zap.L().Error("save inference result failed", zap.Error(err))
			return err
		}
		zap.L().Info("inference result saved", zap.String("device_id", payload.DeviceID), zap.String("file", filePath))
	}

	return nil
}
