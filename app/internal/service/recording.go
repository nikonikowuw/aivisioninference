package service

import (
	"context"
	"fmt"
	"time"

	"github.com/niko-admin/niko-admin/internal/model"
	"github.com/niko-admin/niko-admin/internal/pkg/zlm"
	"github.com/niko-admin/niko-admin/internal/repository"
)

// RecordingService handles recording management.
type RecordingService struct {
	recordingRepo *repository.RecordingRepository
	zlmClient     *zlm.Client
}

// NewRecordingService creates a new RecordingService.
func NewRecordingService(recordingRepo *repository.RecordingRepository, zlmClient *zlm.Client) *RecordingService {
	return &RecordingService{
		recordingRepo: recordingRepo,
		zlmClient:     zlmClient,
	}
}

// StartRecording starts recording for a device stream.
func (s *RecordingService) StartRecording(ctx context.Context, deviceID, app, stream string) error {
	return s.zlmClient.StartRecord(ctx, zlm.RecordRequest{
		Type:          1, // mp4
		Vhost:         "__defaultVhost__",
		App:           app,
		Stream:        stream,
		MaxSecond:     3600,
	})
}

// StopRecording stops recording for a device stream.
func (s *RecordingService) StopRecording(ctx context.Context, app, stream string) error {
	return s.zlmClient.StopRecord(ctx, zlm.RecordRequest{
		Type:   1,
		Vhost:  "__defaultVhost__",
		App:    app,
		Stream: stream,
	})
}

// QueryRecordings queries recordings by device ID and time range.
func (s *RecordingService) QueryRecordings(ctx context.Context, deviceID string, start, end time.Time, page, pageSize int) ([]model.MediaRecording, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.recordingRepo.FindByDeviceIDAndTimeRange(ctx, deviceID, start, end, page, pageSize)
}

// HandleRecordingCallback processes the on_record_mp4 webhook from ZLM.
func (s *RecordingService) HandleRecordingCallback(ctx context.Context, app, stream, filePath, fileName string, fileSize int64, startTime, endTime int64) error {
	recording := &model.MediaRecording{
		App:        app,
		Stream:     stream,
		FileName:   fileName,
		FilePath:   filePath,
		FileSize:   fileSize,
		RecordType: "plan",
		Status:     "completed",
		StartTime:  time.Unix(startTime, 0),
		EndTime:    time.Unix(endTime, 0),
	}
	return s.recordingRepo.Create(ctx, recording)
}

// GetPlaybackURL generates a playback URL for a recording file.
func (s *RecordingService) GetPlaybackURL(ctx context.Context, recordingID string) (string, error) {
	recording, err := s.recordingRepo.FindByID(ctx, recordingID)
	if err != nil {
		return "", fmt.Errorf("recording not found: %w", err)
	}

	// Use ZLM addStreamProxy to serve the recording file as RTSP
	return fmt.Sprintf("rtsp://localhost:554/%s/%s", recording.App, recording.Stream), nil
}

// DeleteRecording deletes a single recording by ID.
func (s *RecordingService) DeleteRecording(ctx context.Context, id string) error {
	return s.recordingRepo.DeleteByID(ctx, id)
}
