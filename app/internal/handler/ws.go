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
	// Allow all origins in development; tighten in production.
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// WSHandler handles WebSocket connection HTTP requests.
type WSHandler struct {
	hub *ws.Hub
	jwt *jwtutil.Manager
}

// NewWSHandler creates a new WSHandler with the given hub and JWT manager.
func NewWSHandler(hub *ws.Hub, jwt *jwtutil.Manager) *WSHandler {
	return &WSHandler{hub: hub, jwt: jwt}
}

// HandleWebSocket upgrades an HTTP connection to WebSocket, authenticates
// the client via a JWT query parameter, and registers the connection with
// the Hub for real-time messaging.
//
// @Summary      WebSocket 连接
// @Description  升级 HTTP 连接为 WebSocket，通过 token query 参数认证
// @Tags         WebSocket
// @Produce      json
// @Param        token  query  string  true  "JWT access token"
// @Success      101    {object}  string
// @Failure      200    {object}  dto.Response
// @Router       /ws [get]
func (h *WSHandler) HandleWebSocket(c *gin.Context) {
	// Extract JWT from query parameter
	tokenString := c.Query("token")
	if tokenString == "" {
		// Also try the "token" query param via the "authorization" header pattern
		// (some WebSocket clients cannot set custom headers)
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

	// Validate JWT
	claims, err := h.jwt.ValidateAccessToken(tokenString)
	if err != nil {
		zap.L().Debug("ws auth failed", zap.Error(err))
		response.Err(c, apperrors.New(apperrors.ErrTokenInvalid, "令牌无效或已过期"))
		c.Abort()
		return
	}

	// Upgrade to WebSocket
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

	// Register the connection with the Hub.
	// The Hub manages read/write pumps and client lifecycle.
	// Hub.HandleConnection creates a Client, registers it, and starts
	// the read/write goroutines.
	h.hub.HandleConnection(conn, claims.UserID)
}
