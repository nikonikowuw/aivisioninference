package handler

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
	"github.com/niko-admin/niko-admin/internal/pkg/response"
	"github.com/niko-admin/niko-admin/internal/service"
)

// MediaRecordingHandler handles recording-related API requests.
type MediaRecordingHandler struct {
	recordingService *service.RecordingService
}

// NewMediaRecordingHandler creates a new MediaRecordingHandler.
func NewMediaRecordingHandler(recordingService *service.RecordingService) *MediaRecordingHandler {
	return &MediaRecordingHandler{recordingService: recordingService}
}

// RegisterRoutes registers recording routes.
func (h *MediaRecordingHandler) RegisterRoutes(r *gin.RouterGroup) {
	{
		r.GET("/recordings", h.ListRecordings)
		r.POST("/recordings/batch-delete", h.BatchDeleteRecordings)
		r.POST("/recordings/:id/playback", h.GetPlaybackURL)
		r.POST("/recordings/start", h.StartRecording)
		r.POST("/recordings/stop", h.StopRecording)
	}
}

// BatchDeleteRecordings deletes multiple recordings by ID.
func (h *MediaRecordingHandler) BatchDeleteRecordings(c *gin.Context) {
	var req struct {
		IDs []string `json:"ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Err(c, badRequestError(c, err))
		return
	}

	deleted, failed := 0, 0
	for _, id := range req.IDs {
		if err := h.recordingService.DeleteRecording(c.Request.Context(), id); err != nil {
			failed++
		} else {
			deleted++
		}
	}
	response.OK(c, gin.H{"deleted": deleted, "failed": failed})
}

// ListRecordings returns recordings filtered by device ID and time range.
func (h *MediaRecordingHandler) ListRecordings(c *gin.Context) {
	deviceID := c.Query("device_id")
	if deviceID == "" {
		response.Err(c, apperrors.New(apperrors.ErrBadRequest, ""))
		return
	}

	startStr := c.DefaultQuery("start", time.Now().AddDate(0, 0, -7).Format(time.RFC3339))
	endStr := c.DefaultQuery("end", time.Now().Format(time.RFC3339))
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	start, err := time.Parse(time.RFC3339, startStr)
	if err != nil {
		response.Err(c, apperrors.New(apperrors.ErrStartTimeFormat, ""))
		return
	}
	end, err := time.Parse(time.RFC3339, endStr)
	if err != nil {
		response.Err(c, apperrors.New(apperrors.ErrEndTimeFormat, ""))
		return
	}

	recordings, total, err := h.recordingService.QueryRecordings(c.Request.Context(), deviceID, start, end, page, pageSize)
	if err != nil {
		zap.L().Error("query recordings failed", zap.Error(err))
		response.Err(c, apperrors.New(apperrors.ErrInternal, ""))
		return
	}

	response.Page(c, recordings, total, page, pageSize)
}

// GetPlaybackURL returns a playback URL for a recording.
func (h *MediaRecordingHandler) GetPlaybackURL(c *gin.Context) {
	recordingID := c.Param("id")

	url, err := h.recordingService.GetPlaybackURL(c.Request.Context(), recordingID)
	if err != nil {
		response.Err(c, apperrors.New(apperrors.ErrNotFound, ""))
		return
	}

	response.OK(c, gin.H{"url": url})
}

// StartRecording manually starts recording for a device.
func (h *MediaRecordingHandler) StartRecording(c *gin.Context) {
	var req struct {
		DeviceID string `json:"device_id" binding:"required"`
		App      string `json:"app" binding:"required"`
		Stream   string `json:"stream" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Err(c, badRequestError(c, err))
		return
	}

	if err := h.recordingService.StartRecording(c.Request.Context(), req.DeviceID, req.App, req.Stream); err != nil {
		zap.L().Error("start recording failed", zap.Error(err))
		response.Err(c, apperrors.New(apperrors.ErrInternal, ""))
		return
	}

	response.OK(c, gin.H{"message": "recording started"})
}

// StopRecording manually stops recording for a device.
func (h *MediaRecordingHandler) StopRecording(c *gin.Context) {
	var req struct {
		App    string `json:"app" binding:"required"`
		Stream string `json:"stream" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Err(c, badRequestError(c, err))
		return
	}

	if err := h.recordingService.StopRecording(c.Request.Context(), req.App, req.Stream); err != nil {
		zap.L().Error("stop recording failed", zap.Error(err))
		response.Err(c, apperrors.New(apperrors.ErrInternal, ""))
		return
	}

	response.OK(c, gin.H{"message": "recording stopped"})
}
