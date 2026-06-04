package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/niko-admin/niko-admin/internal/service"
)

// MediaWebhookHandler handles ZLMediaKit webhooks.
type MediaWebhookHandler struct {
	mediaService *service.MediaService
	sipService   *service.SIPService
}

// NewMediaWebhookHandler creates a new MediaWebhookHandler.
func NewMediaWebhookHandler(mediaService *service.MediaService, sipService *service.SIPService) *MediaWebhookHandler {
	return &MediaWebhookHandler{
		mediaService: mediaService,
		sipService:   sipService,
	}
}

// RegisterRoutes registers ZLM webhook routes.
func (h *MediaWebhookHandler) RegisterRoutes(r *gin.RouterGroup) {
	g := r.Group("/zlm/callback")
	{
		g.POST("/on_publish", h.OnPublish)
		g.POST("/on_play", h.OnPlay)
		g.POST("/on_stream_changed", h.OnStreamChanged)
		g.POST("/on_stream_not_found", h.OnStreamNotFound)
		g.POST("/on_record_mp4", h.OnRecordMP4)
		g.POST("/on_server_started", h.OnServerStarted)
		g.POST("/on_register", h.OnRegister)
	}
}

// WebhookRsp is the standard success response for ZLM webhooks.
type WebhookRsp struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

func successWebhook(c *gin.Context) {
	c.JSON(http.StatusOK, WebhookRsp{Code: 0, Msg: "success"})
}

func (h *MediaWebhookHandler) bindJSON(c *gin.Context, req interface{}) bool {
	if err := c.ShouldBindJSON(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "msg": "invalid params"})
		return false
	}
	return true
}

// OnRegister handles SIP device registration.
func (h *MediaWebhookHandler) OnRegister(c *gin.Context) {
	var req struct {
		DeviceID string `json:"device_id"`
		RemoteIP string `json:"remote_ip"`
		Port     int    `json:"port"`
	}
	if !h.bindJSON(c, &req) {
		return
	}

	zap.L().Info("ZLM on_register received", zap.String("device_id", req.DeviceID), zap.String("ip", req.RemoteIP))

	// TODO: 调用 sipService 验证设备
	// if err := h.sipService.HandleRegister(c.Request.Context(), req.DeviceID, req.RemoteIP, req.Port); err != nil {
	// 	c.JSON(http.StatusOK, gin.H{"code": -1, "msg": err.Error()})
	// 	return
	// }

	successWebhook(c)
}

// OnPublish handles stream publishing event.
func (h *MediaWebhookHandler) OnPublish(c *gin.Context) {
	var req struct {
		App    string `json:"app"`
		Stream string `json:"stream"`
		Vhost  string `json:"vhost"`
		Schema string `json:"schema"`
	}
	if !h.bindJSON(c, &req) {
		return
	}

	zap.L().Info("ZLM on_publish received", zap.String("app", req.App), zap.String("stream", req.Stream))

	// TODO: 调用 mediaService 更新流状态
	// h.mediaService.UpdateStreamStatus(c.Request.Context(), req.App, req.Stream, "active")

	successWebhook(c)
}

// OnPlay handles playback request authentication.
func (h *MediaWebhookHandler) OnPlay(c *gin.Context) {
	var req struct {
		App    string `json:"app"`
		Stream string `json:"stream"`
		Params string `json:"params"`
	}
	if !h.bindJSON(c, &req) {
		return
	}

	zap.L().Info("ZLM on_play received", zap.String("app", req.App), zap.String("stream", req.Stream), zap.String("params", req.Params))

	// TODO: 调用 mediaService 验证播放权限
	// if err := h.mediaService.VerifyPlayAuth(c.Request.Context(), req.App, req.Stream, req.Params); err != nil {
	// 	c.JSON(http.StatusOK, gin.H{"code": -1, "msg": "auth failed"})
	// 	return
	// }

	successWebhook(c)
}

// OnStreamChanged handles stream status change event.
func (h *MediaWebhookHandler) OnStreamChanged(c *gin.Context) {
	var req struct {
		App    string `json:"app"`
		Stream string `json:"stream"`
		Vhost  string `json:"vhost"`
		Schema string `json:"schema"`
		Status int    `json:"status"`
	}
	if !h.bindJSON(c, &req) {
		return
	}

	zap.L().Info("ZLM on_stream_changed", zap.String("app", req.App), zap.String("stream", req.Stream), zap.Int("status", req.Status))

	// TODO: 更新媒体流状态

	successWebhook(c)
}

// OnStreamNotFound handles stream not found event.
func (h *MediaWebhookHandler) OnStreamNotFound(c *gin.Context) {
	var req struct {
		App    string `json:"app"`
		Stream string `json:"stream"`
		Vhost  string `json:"vhost"`
		Schema string `json:"schema"`
	}
	if !h.bindJSON(c, &req) {
		return
	}

	zap.L().Warn("ZLM on_stream_not_found", zap.String("app", req.App), zap.String("stream", req.Stream))

	// TODO: 更新流状态为 error，触发重试

	successWebhook(c)
}

// OnRecordMP4 handles mp4 recording completion event.
func (h *MediaWebhookHandler) OnRecordMP4(c *gin.Context) {
	var req struct {
		App       string `json:"app"`
		Stream    string `json:"stream"`
		FileName  string `json:"file_name"`
		FilePath  string `json:"file_path"`
		FileSize  int64  `json:"file_size"`
		StartTime int64  `json:"start_time"`
		EndTime   int64  `json:"end_time"`
	}
	if !h.bindJSON(c, &req) {
		return
	}

	zap.L().Info("ZLM on_record_mp4", zap.String("app", req.App), zap.String("stream", req.Stream), zap.String("file_path", req.FilePath))

	// TODO: 建立录像索引

	successWebhook(c)
}

// OnServerStarted handles ZLM server start event.
func (h *MediaWebhookHandler) OnServerStarted(c *gin.Context) {
	var req struct {
		Version string `json:"version"`
	}
	if !h.bindJSON(c, &req) {
		return
	}

	zap.L().Info("ZLM server started", zap.String("version", req.Version))

	// TODO: 恢复 ZLM 重启前的流

	successWebhook(c)
}
