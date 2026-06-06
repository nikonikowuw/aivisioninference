package handler

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/niko-admin/niko-admin/internal/pkg/response"
	"github.com/niko-admin/niko-admin/internal/service"
)

type MediaGB28181Handler struct {
	sipSvc *service.SIPService
}

func NewMediaGB28181Handler(sipSvc *service.SIPService) *MediaGB28181Handler {
	return &MediaGB28181Handler{sipSvc: sipSvc}
}

type StartLiveRequest struct {
	DeviceID string `json:"device_id" binding:"required"`
}

type StopLiveRequest struct {
	DeviceID string `json:"device_id" binding:"required"`
	StreamID string `json:"stream_id" binding:"required"`
}

type PlayResponse struct {
	URL      string `json:"url"`
	Protocol string `json:"protocol"`
	StreamID string `json:"stream_id"`
}

// StartLive 启动 GB28181 实时预览
// @Summary      启动实时预览
// @Tags         GB28181媒体
// @Accept       json
// @Produce      json
// @Param        body  body  StartLiveRequest  true  "请求参数"
// @Success      200  {object}  dto.Response{data=PlayResponse}
// @Router       /media/gb28181/live/start [post]
// @Security     BearerAuth
func (h *MediaGB28181Handler) StartLive(c *gin.Context) {
	var req StartLiveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		attachError(c, badRequestError(c, err))
		return
	}

	gbDevice, err := h.sipSvc.GetGB28181DeviceByID(c.Request.Context(), req.DeviceID)
	if err != nil {
		attachError(c, err)
		return
	}

	streamID := uuid.New().String()
	url, err := h.sipSvc.StartLiveStream(c.Request.Context(), gbDevice.DeviceCode, streamID)
	if err != nil {
		attachError(c, err)
		return
	}

	response.OK(c, PlayResponse{
		URL:      url,
		Protocol: "rtmp",
		StreamID: streamID,
	})
}

// StopLive 停止 GB28181 实时预览
// @Summary      停止实时预览
// @Tags         GB28181媒体
// @Accept       json
// @Produce      json
// @Param        body  body  StopLiveRequest  true  "请求参数"
// @Success      200  {object}  dto.Response
// @Router       /media/gb28181/live/stop [post]
// @Security     BearerAuth
func (h *MediaGB28181Handler) StopLive(c *gin.Context) {
	var req StopLiveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		attachError(c, badRequestError(c, err))
		return
	}

	gbDevice, err := h.sipSvc.GetGB28181DeviceByID(c.Request.Context(), req.DeviceID)
	if err != nil {
		attachError(c, err)
		return
	}

	if err := h.sipSvc.StopLiveStream(c.Request.Context(), gbDevice.DeviceCode, req.StreamID); err != nil {
		attachError(c, err)
		return
	}

	response.OK(c, nil)
}

type StartPlaybackRequest struct {
	DeviceID  string    `json:"device_id" binding:"required"`
	StartTime time.Time `json:"start_time" binding:"required"`
	EndTime   time.Time `json:"end_time" binding:"required"`
}

// StartPlayback 启动 GB28181 录像回放
// @Summary      启动录像回放
// @Tags         GB28181媒体
// @Accept       json
// @Produce      json
// @Param        body  body  StartPlaybackRequest  true  "请求参数"
// @Success      200  {object}  dto.Response{data=PlayResponse}
// @Router       /media/gb28181/playback/start [post]
// @Security     BearerAuth
func (h *MediaGB28181Handler) StartPlayback(c *gin.Context) {
	var req StartPlaybackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		attachError(c, badRequestError(c, err))
		return
	}

	gbDevice, err := h.sipSvc.GetGB28181DeviceByID(c.Request.Context(), req.DeviceID)
	if err != nil {
		attachError(c, err)
		return
	}

	streamID := uuid.New().String()
	url, err := h.sipSvc.StartPlayback(c.Request.Context(), gbDevice.DeviceCode, streamID, req.StartTime, req.EndTime)
	if err != nil {
		attachError(c, err)
		return
	}

	response.OK(c, PlayResponse{
		URL:      url,
		Protocol: "flv",
		StreamID: streamID,
	})
}

type PlaybackControlRequest struct {
	StreamID string  `json:"stream_id" binding:"required"`
	Action   string  `json:"action" binding:"required,oneof=pause resume scale seek"`
	Speed    float64 `json:"speed"`
	Stamp    int64   `json:"stamp"`
}

// PlaybackControl GB28181 回放控制
// @Summary      回放控制
// @Tags         GB28181媒体
// @Accept       json
// @Produce      json
// @Param        body  body  PlaybackControlRequest  true  "控制参数"
// @Success      200  {object}  dto.Response
// @Router       /media/gb28181/playback/control [post]
// @Security     BearerAuth
func (h *MediaGB28181Handler) PlaybackControl(c *gin.Context) {
	var req PlaybackControlRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		attachError(c, badRequestError(c, err))
		return
	}

	if err := h.sipSvc.PlaybackControl(c.Request.Context(), req.StreamID, req.Action, req.Speed, req.Stamp); err != nil {
		attachError(c, err)
		return
	}

	response.OK(c, nil)
}

type StopPlaybackRequest struct {
	DeviceID string `json:"device_id" binding:"required"`
	StreamID string `json:"stream_id" binding:"required"`
}

// StopPlayback 停止 GB28181 录像回放
// @Summary      停止录像回放
// @Tags         GB28181媒体
// @Accept       json
// @Produce      json
// @Param        body  body  StopPlaybackRequest  true  "请求参数"
// @Success      200  {object}  dto.Response
// @Router       /media/gb28181/playback/stop [post]
// @Security     BearerAuth
func (h *MediaGB28181Handler) StopPlayback(c *gin.Context) {
	var req StopPlaybackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		attachError(c, badRequestError(c, err))
		return
	}

	gbDevice, err := h.sipSvc.GetGB28181DeviceByID(c.Request.Context(), req.DeviceID)
	if err != nil {
		attachError(c, err)
		return
	}

	// 停止回放也是调用 ZLM close stream，可以用 StopLiveStream，或提供专用的 StopPlayback
	// 这里复用 StopLiveStream，效果是一样的：停止 rtp 接收
	if err := h.sipSvc.StopLiveStream(c.Request.Context(), gbDevice.DeviceCode, req.StreamID); err != nil {
		attachError(c, err)
		return
	}

	response.OK(c, nil)
}
