package task

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/hibiken/asynq"
	"github.com/niko-admin/niko-admin/internal/pkg/zlm"
	"go.uber.org/zap"
)

const (
	// TypeTimedScreenshot is the Asynq task type for timed screenshot capture
	TypeTimedScreenshot = "ai:timed_screenshot"
)

// TimedScreenshotPayload is the payload for timed screenshot tasks.
type TimedScreenshotPayload struct {
	DeviceID   string `json:"device_id"`
	StreamURL  string `json:"stream_url"`
	TaskID     string `json:"task_id"`
	Interval   int    `json:"interval"`   // seconds between captures
	Algorithms string `json:"algorithms"` // comma-separated algorithm list
}

// ScreenshotStorage handles saving screenshot files.
type ScreenshotStorage struct {
	BasePath string
}

// NewScreenshotStorage creates a new ScreenshotStorage.
func NewScreenshotStorage(basePath string) *ScreenshotStorage {
	if basePath == "" {
		basePath = "./data/snapshots"
	}
	return &ScreenshotStorage{BasePath: basePath}
}

// Save saves screenshot bytes to disk and returns the file path and URL.
func (s *ScreenshotStorage) Save(deviceID string, data []byte, timestamp int64) (string, error) {
	dir := filepath.Join(s.BasePath, deviceID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("create snapshot dir: %w", err)
	}

	filename := fmt.Sprintf("%d.jpg", timestamp)
	filePath := filepath.Join(dir, filename)

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return "", fmt.Errorf("write snapshot file: %w", err)
	}

	return filePath, nil
}

// HandleTimedScreenshot processes a timed screenshot task.
func HandleTimedScreenshot(ctx context.Context, t *asynq.Task, zlmClient *zlm.Client, storage *ScreenshotStorage) error {
	var payload TimedScreenshotPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal payload: %w", err)
	}

	zap.L().Info("processing timed screenshot",
		zap.String("device_id", payload.DeviceID),
		zap.String("task_id", payload.TaskID),
	)

	// Capture snapshot from ZLM
	imgData, err := zlmClient.GetSnap(ctx, zlm.GetSnapRequest{
		URL:        payload.StreamURL,
		TimeoutSec: 10,
		ExpireSec:  30,
	})
	if err != nil {
		zap.L().Error("timed snapshot failed", zap.String("device_id", payload.DeviceID), zap.Error(err))
		return fmt.Errorf("zlm snapshot: %w", err)
	}

	// Save to storage
	now := time.Now().Unix()
	filePath, err := storage.Save(payload.DeviceID, imgData, now)
	if err != nil {
		zap.L().Error("save snapshot failed", zap.Error(err))
		return err
	}

	zap.L().Info("timed screenshot saved",
		zap.String("device_id", payload.DeviceID),
		zap.String("path", filePath),
		zap.Int("size", len(imgData)),
	)

	// TODO: 调用 AI 推理 API
	// 1. Send filePath to AI inference service
	// 2. Save inference results to database
	// 3. Trigger alert if needed

	return nil
}
