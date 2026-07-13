package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/service"
)

// EdgeNodeTerminalHandler handles WebSocket terminal connections.
type EdgeNodeTerminalHandler struct {
	terminalSvc    *service.EdgeNodeTerminalService
	allowedOrigins []string
}

// NewEdgeNodeTerminalHandler creates a new terminal handler.
func NewEdgeNodeTerminalHandler(terminalSvc *service.EdgeNodeTerminalService, allowedOrigins []string) *EdgeNodeTerminalHandler {
	return &EdgeNodeTerminalHandler{
		terminalSvc:    terminalSvc,
		allowedOrigins: allowedOrigins,
	}
}

var terminalUpgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
}

// HandleWebSocket handles WebSocket upgrade and terminal session lifecycle.
//
// @Summary      Web终端 WebSocket
// @Description  通过 WebSocket 连接到边缘节点的远程终端
// @Tags         边缘节点
// @Param        id  path  string  true  "节点 ID"
// @Success      101  "Switching Protocols"
// @Router       /edge-nodes/{id}/terminal [get]
// @Security     BearerAuth
func (h *EdgeNodeTerminalHandler) HandleWebSocket(c *gin.Context) {
	nodeID := c.Param("id")
	if nodeID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少节点 ID"})
		return
	}

	// Per-request CheckOrigin using the configured CORS whitelist.
	terminalUpgrader.CheckOrigin = func(r *http.Request) bool {
		return isWebSocketOriginAllowed(r, h.allowedOrigins)
	}

	// Upgrade HTTP to WebSocket
	conn, err := terminalUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		zap.L().Error("WebSocket upgrade failed",
			zap.String("node_id", nodeID),
			zap.Error(err),
		)
		return
	}

	// Serialize writes to the WebSocket connection.
	var writeMu sync.Mutex
	writeJSON := func(msg interface{}) error {
		writeMu.Lock()
		defer writeMu.Unlock()
		return conn.WriteJSON(msg)
	}

	// Open terminal session
	sessionID, err := h.terminalSvc.OpenSession(c.Request.Context(), nodeID, func(data []byte) {
		_ = writeJSON(json.RawMessage(data))
	})
	if err != nil {
		zap.L().Error("failed to open terminal session",
			zap.String("node_id", nodeID),
			zap.Error(err),
		)
		_ = conn.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseInternalServerErr, "failed to open terminal session"))
		_ = conn.Close()
		return
	}

	// Send session ID to client
	_ = writeJSON(dto.WebSocketTerminalMessage{
		Type:      "session",
		SessionID: sessionID,
		Data:      "Session opened",
	})

	zap.L().Info("Web Terminal session established",
		zap.String("session_id", sessionID),
		zap.String("node_id", nodeID),
	)

	// Read loop: receive messages from WS, forward to MQTT.
	// Each iteration creates a fresh context with timeout so a hung MQTT
	// publish on one message doesn't block subsequent messages indefinitely.
	done := make(chan struct{})
	go func() {
		defer close(done)

		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure, websocket.CloseNoStatusReceived) {
					zap.L().Debug("WebSocket read error",
						zap.String("session_id", sessionID),
						zap.Error(err),
					)
				}
				return
			}

			var msg dto.WebSocketTerminalMessage
			if err := json.Unmarshal(message, &msg); err != nil {
				zap.L().Debug("invalid terminal message format",
					zap.String("session_id", sessionID),
					zap.Error(err),
				)
				continue
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

			switch msg.Type {
			case "input":
				// Send terminal input to engine PTY
				if err := h.terminalSvc.WriteToSession(ctx, sessionID, msg.Data); err != nil {
					zap.L().Error("failed to write to terminal session",
						zap.String("session_id", sessionID),
						zap.Error(err),
					)
					_ = writeJSON(dto.WebSocketTerminalMessage{
						Type:  "error",
						Error: err.Error(),
					})
				}

			case "resize":
				// Resize terminal window
				if err := h.terminalSvc.ResizeSession(ctx, sessionID, msg.Cols, msg.Rows); err != nil {
					zap.L().Error("failed to resize terminal session",
						zap.String("session_id", sessionID),
						zap.Error(err),
					)
				}

			case "close":
				cancel()
				_ = h.terminalSvc.CloseSession(ctx, sessionID)
				return

			case "ping":
				_ = writeJSON(dto.WebSocketTerminalMessage{
					Type: "pong",
				})

			default:
				zap.L().Debug("unknown terminal message type",
					zap.String("type", msg.Type),
				)
			}

			cancel()
		}
	}()

	// Wait for the read goroutine to exit (client disconnect or error).
	// Idle timeout is enforced by EdgeNodeTerminalService.cleanupLoop
	// (default 5 min), not here, so active sessions are never force-killed.
	<-done

	// Close session with a short deadline for MQTT publish.
	closeCtx, closeCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer closeCancel()
	_ = h.terminalSvc.CloseSession(closeCtx, sessionID)
	_ = conn.Close()

	zap.L().Info("Web Terminal session ended",
		zap.String("session_id", sessionID),
		zap.String("node_id", nodeID),
	)
}
