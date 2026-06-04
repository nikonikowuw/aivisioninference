package router

import (
	"github.com/gin-gonic/gin"

	"github.com/niko-admin/niko-admin/internal/handler"
	"github.com/niko-admin/niko-admin/internal/middleware"
	"github.com/niko-admin/niko-admin/internal/pkg/jwt"
)

// MediaRouter registers media-related API routes.
type MediaRouter struct {
	webhookHandler   *handler.MediaWebhookHandler
	mediaPlayHandler *handler.MediaPlayHandler
	recordingHandler *handler.MediaRecordingHandler
	jwtManager       *jwt.Manager
}

// NewMediaRouter creates a new MediaRouter.
func NewMediaRouter(
	webhookHandler *handler.MediaWebhookHandler,
	mediaPlayHandler *handler.MediaPlayHandler,
	recordingHandler *handler.MediaRecordingHandler,
	jwtManager *jwt.Manager,
) *MediaRouter {
	return &MediaRouter{
		webhookHandler:   webhookHandler,
		mediaPlayHandler: mediaPlayHandler,
		recordingHandler: recordingHandler,
		jwtManager:       jwtManager,
	}
}

// RegisterRoutes registers the media route groups.
func (r *MediaRouter) RegisterRoutes(router *gin.Engine) {
	// ZLM Webhook callbacks - 需要 ZLMWebhookAuth 中间件
	r.webhookHandler.RegisterRoutes(&router.RouterGroup)

	// Media playback API - 需要 JWT 鉴权
	apiGroup := router.Group("/api/v1/media")
	apiGroup.Use(middleware.Auth(r.jwtManager))
	r.mediaPlayHandler.RegisterRoutes(apiGroup)
	r.recordingHandler.RegisterRoutes(apiGroup)
}
