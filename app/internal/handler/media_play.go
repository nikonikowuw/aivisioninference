package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
	"github.com/niko-admin/niko-admin/internal/pkg/response"
	"github.com/niko-admin/niko-admin/internal/service"
	"go.uber.org/zap"
)

// MediaPlayHandler handles playback-related API requests.
type MediaPlayHandler struct {
	mediaService *service.MediaService
}

// NewMediaPlayHandler creates a new MediaPlayHandler.
func NewMediaPlayHandler(mediaService *service.MediaService) *MediaPlayHandler {
	return &MediaPlayHandler{mediaService: mediaService}
}

// RegisterRoutes registers media playback routes on the given media sub-group.
func (h *MediaPlayHandler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/play", h.GetPlayURL)
	r.POST("/stop", h.StopPlay)
	r.GET("/snapshot", h.GetSnapshot)
	r.GET("/streams", h.ListStreams)
}

// GetPlayURL returns a signed playback URL for the specified device and protocol.
// 对于 RTSP 设备，会自动通过 ZLM addStreamProxy 按需拉流。
func (h *MediaPlayHandler) GetPlayURL(c *gin.Context) {
	deviceID := c.Query("device_id")
	protocol := c.DefaultQuery("protocol", "auto")
	streamType := c.DefaultQuery("stream_type", "main")

	if deviceID == "" {
		response.Err(c, apperrors.New(apperrors.ErrBadRequest, ""))
		return
	}

	url, codec, err := h.mediaService.GetPlayURL(c.Request.Context(), deviceID, protocol, streamType)
	if err != nil {
		zap.L().Error("get play url failed", zap.String("device_id", deviceID), zap.Error(err))
		response.Err(c, apperrors.New(apperrors.ErrInternal, ""))
		return
	}

	response.OK(c, gin.H{
		"url":         url,
		"protocol":    protocol,
		"stream_type": streamType,
		"codec":       codec,
		"expires":     time.Now().Add(30 * time.Minute).Unix(),
	})
}

// StopPlay 停止设备的流代理（关闭预览）。
func (h *MediaPlayHandler) StopPlay(c *gin.Context) {
	deviceID := c.Query("device_id")
	streamType := c.DefaultQuery("stream_type", "main")
	if deviceID == "" {
		response.Err(c, apperrors.New(apperrors.ErrBadRequest, ""))
		return
	}

	if err := h.mediaService.StopPlayURL(c.Request.Context(), deviceID, streamType); err != nil {
		zap.L().Warn("stop play failed", zap.String("device_id", deviceID), zap.Error(err))
		// 关闭失败不是致命错误
	}

	response.OK(c, gin.H{"message": "stopped"})
}

// GetSnapshot captures a snapshot from the device stream.
func (h *MediaPlayHandler) GetSnapshot(c *gin.Context) {
	deviceID := c.Query("device_id")
	if deviceID == "" {
		response.Err(c, apperrors.New(apperrors.ErrBadRequest, ""))
		return
	}

	imgData, err := h.mediaService.GetSnapshot(c.Request.Context(), deviceID)
	if err != nil {
		zap.L().Error("get snapshot failed", zap.String("device_id", deviceID), zap.Error(err))
		response.Err(c, apperrors.New(apperrors.ErrInternal, ""))
		return
	}

	c.Data(http.StatusOK, "image/jpeg", imgData)
}

// ListStreams returns all active streams in StreamManager.
func (h *MediaPlayHandler) ListStreams(c *gin.Context) {
	streams := h.mediaService.ListStreams(c.Request.Context())
	response.OK(c, streams)
}
