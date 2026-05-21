package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
	jwtutil "github.com/niko-admin/niko-admin/internal/pkg/jwt"
	"github.com/niko-admin/niko-admin/internal/pkg/response"
	"github.com/niko-admin/niko-admin/internal/pkg/ws"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// 开发环境允许所有跨域请求；生产环境需进行严格域名限制。
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// WSHandler 处理 WebSocket 连接升级的 HTTP 请求。
type WSHandler struct {
	hub *ws.Hub
	jwt *jwtutil.Manager
}

// NewWSHandler 创建一个新的 WSHandler 实例。
func NewWSHandler(hub *ws.Hub, jwt *jwtutil.Manager) *WSHandler {
	return &WSHandler{hub: hub, jwt: jwt}
}

// HandleWebSocket 将 HTTP 连接升级为 WebSocket 协议，通过 token 查询参数进行 JWT 身份验证，并将连接注册到 Hub 中心以实现实时消息推送。
//
// @Summary      WebSocket 连接
// @Description  升级 HTTP 连接为 WebSocket，通过 token query 参数认证
// @Tags         WebSocket
// @Produce      json
// @Param        token  query  string  true  "JWT access token"
// @Success      101    {object}  string
// @Failure      401    {object}  dto.Response
// @Router       /ws [get]
func (h *WSHandler) HandleWebSocket(c *gin.Context) {
	// 从查询参数或 Authorization 请求头中提取 JWT 访问令牌
	tokenString := c.Query("token")
	if tokenString == "" {
		// 如果 query 中没有 token，尝试从 Authorization 头中解析 Bearer token
		//（部分 WebSocket 客户端不支持在建立握手时直接设置自定义 Header，通常会写在 URL 传参中）
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			tokenString = authHeader[7:]
		}
	}

	if tokenString == "" {
		response.Err(c, apperrors.New(apperrors.ErrUnauthorized, "缺少认证令牌"))
		c.Abort()
		return
	}

	// 验证 JWT 访问令牌的有效性
	claims, err := h.jwt.ValidateAccessToken(tokenString)
	if err != nil {
		zap.L().Debug("ws auth failed", zap.Error(err))
		response.Err(c, apperrors.New(apperrors.ErrTokenInvalid, "令牌无效或已过期"))
		c.Abort()
		return
	}

	// 升级当前的 HTTP 协议连接为 WebSocket 协议
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		zap.L().Error("websocket upgrade failed",
			zap.String("user_id", claims.UserID),
			zap.Error(err),
		)
		return
	}

	zap.L().Info("websocket connected",
		zap.String("user_id", claims.UserID),
		zap.String("remote_addr", conn.RemoteAddr().String()),
	)

	// 将该 WebSocket 连接注册到全局 Hub。Hub 将管理消息的读/写通道及客户端生命周期。
	// Hub.HandleConnection 将会创建一个 Client 结构体并对其进行异步监听读写事件。
	h.hub.HandleConnection(conn, claims.UserID)
}
