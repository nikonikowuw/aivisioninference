package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/niko-admin/niko-admin/internal/model"
	"github.com/niko-admin/niko-admin/internal/service"
)

// MediaWebhookHandler handles ZLMediaKit webhooks.
type MediaWebhookHandler struct {
	mediaService   *service.MediaService
	sipService     *service.SIPService
	streamManager  *service.StreamManager
	stagingService *service.DeviceStagingService
}

// NewMediaWebhookHandler creates a new MediaWebhookHandler.
func NewMediaWebhookHandler(
	mediaService *service.MediaService,
	sipService *service.SIPService,
	streamManager *service.StreamManager,
	stagingService *service.DeviceStagingService,
) *MediaWebhookHandler {
	return &MediaWebhookHandler{
		mediaService:   mediaService,
		sipService:     sipService,
		streamManager:  streamManager,
		stagingService: stagingService,
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
		g.POST("/on_catalog", h.OnCatalogResponse)
		g.POST("/on_alarm", h.OnAlarm)
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

	// 1. 尝试调用 sipService 验证设备并注册
	if h.sipService != nil {
		err := h.sipService.HandleRegister(c.Request.Context(), req.DeviceID, req.RemoteIP, req.Port)
		if err != nil {
			// 如果设备未预注册，记录到发现暂存区
			zap.L().Warn("device not pre-registered, adding to staging", zap.String("device_id", req.DeviceID), zap.Error(err))

			if stagingErr := h.stagingService.AddDiscovered(c.Request.Context(), &model.DiscoveredDevice{
				Source:      model.SourceGB28181,
				DeviceIP:    req.RemoteIP,
				GB28181Code: req.DeviceID,
				Status:      model.StatusPending,
			}); stagingErr != nil {
				zap.L().Error("failed to add discovered device to staging",
					zap.String("device_id", req.DeviceID), zap.Error(stagingErr))
			}
		}
	}

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

	// 通知 StreamManager
	h.streamManager.HandleStreamOnline(c.Request.Context(), req.App, req.Stream, req.Vhost)

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

	zap.L().Info("ZLM on_play received", zap.String("app", req.App), zap.String("stream", req.Stream))

	if err := h.streamManager.VerifyPlaybackAuth(c.Request.Context(), req.App, req.Stream, req.Params); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": -1, "msg": "auth failed"})
		return
	}

	successWebhook(c)
}

// OnStreamChanged handles stream status change event.
func (h *MediaWebhookHandler) OnStreamChanged(c *gin.Context) {
	var req struct {
		App    string `json:"app"`
		Stream string `json:"stream"`
		Vhost  string `json:"vhost"`
		Schema string `json:"schema"`
		Status int    `json:"status"` // 1: 注册, 0: 注销
	}
	if !h.bindJSON(c, &req) {
		return
	}

	zap.L().Info("ZLM on_stream_changed", zap.String("app", req.App), zap.String("stream", req.Stream), zap.Int("status", req.Status))

	if req.Status == 1 {
		h.streamManager.HandleStreamOnline(c.Request.Context(), req.App, req.Stream, req.Vhost)
	} else {
		h.streamManager.HandleStreamOffline(c.Request.Context(), req.App, req.Stream, req.Vhost)
	}

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

	// 触发 StreamManager 的重试逻辑
	h.streamManager.HandleStreamNotFound(c.Request.Context(), req.App, req.Stream, req.Vhost)

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

	zap.L().Info("ZLM on_record_mp4 received", 
		zap.String("stream", req.Stream), 
		zap.String("path", req.FilePath))

	// TODO: 真正的数据库入库逻辑
	// h.mediaService.CreateRecordingRecord(c.Request.Context(), req)

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

	// 恢复所有流
	h.streamManager.RecoverOnServerStart(c.Request.Context())

	successWebhook(c)
}

// OnCatalogResponse handles GB28181 device catalog response forwarded by ZLM.
// ZLM 接收到设备的目录响应后会通过此 webhook 转发给 Go 控制面。
func (h *MediaWebhookHandler) OnCatalogResponse(c *gin.Context) {
	var req struct {
		DeviceID  string `json:"device_id"`  // NVR 设备国标编码
		XMLData   string `json:"xml"`        // 原始 MANSCDP XML
		Channel   string `json:"channel"`     // 通道名（备用）
	}
	if !h.bindJSON(c, &req) {
		return
	}

	zap.L().Info("ZLM on_catalog received", zap.String("device_id", req.DeviceID))

	if h.sipService == nil || req.XMLData == "" {
		successWebhook(c)
		return
	}

	// 解析 MANSCDP 目录响应
	channels, err := service.ParseCatalogueResponse(req.XMLData)
	if err != nil {
		zap.L().Warn("parse catalog response failed",
			zap.String("device_id", req.DeviceID), zap.Error(err))
		successWebhook(c)
		return
	}

	// 同步到 Device 表
	if err := h.sipService.SyncCatalogChannels(c.Request.Context(), req.DeviceID, channels); err != nil {
		zap.L().Error("sync catalog channels failed",
			zap.String("device_id", req.DeviceID), zap.Error(err))
	}

	successWebhook(c)
}

// OnAlarm handles GB28181 alarm notification forwarded by ZLM.
func (h *MediaWebhookHandler) OnAlarm(c *gin.Context) {
	var req struct {
		DeviceID string `json:"device_id"`
		XMLData  string `json:"xml"`
	}
	if !h.bindJSON(c, &req) {
		return
	}

	zap.L().Info("ZLM on_alarm received", zap.String("device_id", req.DeviceID))

	if h.sipService == nil || req.XMLData == "" {
		successWebhook(c)
		return
	}

	// 解析 MANSCDP 告警
	alarm, err := service.ParseAlarmResponse(req.XMLData)
	if err != nil {
		zap.L().Warn("parse alarm response failed",
			zap.String("device_id", req.DeviceID), zap.Error(err))
		successWebhook(c)
		return
	}
	// DeviceID 可能来自 payload 或 XML 内部，优先使用 XML 中的
	if alarm.DeviceID == "" {
		alarm.DeviceID = req.DeviceID
	}

	// 处理告警（写入 smart_records + 推入 Asynq 告警分发队列）
	if err := h.sipService.HandleAlarm(c.Request.Context(), *alarm); err != nil {
		zap.L().Error("handle alarm failed",
			zap.String("device_id", alarm.DeviceID), zap.Error(err))
	}

	successWebhook(c)
}
