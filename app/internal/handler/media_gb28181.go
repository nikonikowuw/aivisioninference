package handler

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/pkg/response"
	"github.com/niko-admin/niko-admin/internal/pkg/zlm"
	"github.com/niko-admin/niko-admin/internal/service"
)

type MediaGB28181Handler struct {
	sipSvc *service.SIPService
}

func NewMediaGB28181Handler(sipSvc *service.SIPService) *MediaGB28181Handler {
	return &MediaGB28181Handler{sipSvc: sipSvc}
}

// StartLive 启动 GB28181 实时预览
// @Summary      启动实时预览
// @Tags         GB28181媒体
// @Accept       json
// @Produce      json
// @Param        body  body  dto.StartLiveRequest  true  "请求参数"
// @Success      200  {object}  dto.Response{data=dto.PlayResponse}
// @Router       /media/gb28181/live/start [post]
// @Security     BearerAuth
func (h *MediaGB28181Handler) StartLive(c *gin.Context) {
	var req dto.StartLiveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		attachError(c, badRequestError(c, err))
		return
	}

	deviceCode, err := h.sipSvc.ResolveGB28181DeviceCode(c.Request.Context(), req.DeviceID)
	if err != nil {
		attachError(c, err)
		return
	}

	streamID := uuid.New().String()
	url, err := h.sipSvc.StartLiveStream(c.Request.Context(), deviceCode, streamID)
	if err != nil {
		attachError(c, err)
		return
	}

	response.OK(c, dto.PlayResponse{
		URL:      url,
		Protocol: zlm.ProtocolWebRTC,
		StreamID: streamID,
	})
}

// StopLive 停止 GB28181 实时预览
// @Summary      停止实时预览
// @Tags         GB28181媒体
// @Accept       json
// @Produce      json
// @Param        body  body  dto.StopLiveRequest  true  "请求参数"
// @Success      200  {object}  dto.Response
// @Router       /media/gb28181/live/stop [post]
// @Security     BearerAuth
func (h *MediaGB28181Handler) StopLive(c *gin.Context) {
	var req dto.StopLiveRequest
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

// StartPlayback 启动 GB28181 录像回放
// @Summary      启动录像回放
// @Tags         GB28181媒体
// @Accept       json
// @Produce      json
// @Param        body  body  dto.StartPlaybackRequest  true  "请求参数"
// @Success      200  {object}  dto.Response{data=dto.PlayResponse}
// @Router       /media/gb28181/playback/start [post]
// @Security     BearerAuth
func (h *MediaGB28181Handler) StartPlayback(c *gin.Context) {
	var req dto.StartPlaybackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		attachError(c, badRequestError(c, err))
		return
	}

	gbDevice, err := h.sipSvc.GetGB28181DeviceByID(c.Request.Context(), req.DeviceID)
	if err != nil {
		attachError(c, err)
		return
	}

	streamID := fmt.Sprintf("playback_%s_%s", gbDevice.DeviceCode, uuid.New().String()[:8])
	url, err := h.sipSvc.StartPlayback(c.Request.Context(), gbDevice.DeviceCode, streamID, req.StartTime, req.EndTime)
	if err != nil {
		attachError(c, err)
		return
	}

	response.OK(c, dto.PlayResponse{
		URL:      url,
		Protocol: zlm.ProtocolFLV,
		StreamID: streamID,
	})
}

// PlaybackControl GB28181 回放控制
// @Summary      回放控制
// @Tags         GB28181媒体
// @Accept       json
// @Produce      json
// @Param        body  body  dto.PlaybackControlRequest  true  "控制参数"
// @Success      200  {object}  dto.Response
// @Router       /media/gb28181/playback/control [post]
// @Security     BearerAuth
func (h *MediaGB28181Handler) PlaybackControl(c *gin.Context) {
	var req dto.PlaybackControlRequest
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

// StopPlayback 停止 GB28181 录像回放
// @Summary      停止录像回放
// @Tags         GB28181媒体
// @Accept       json
// @Produce      json
// @Param        body  body  dto.StopPlaybackRequest  true  "请求参数"
// @Success      200  {object}  dto.Response
// @Router       /media/gb28181/playback/stop [post]
// @Security     BearerAuth
func (h *MediaGB28181Handler) StopPlayback(c *gin.Context) {
	var req dto.StopPlaybackRequest
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
