package handler

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	gossh "golang.org/x/crypto/ssh"
	"go.uber.org/zap"

	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
	"github.com/niko-admin/niko-admin/internal/pkg/jwt"
	"github.com/niko-admin/niko-admin/internal/pkg/response"
	"github.com/niko-admin/niko-admin/internal/service"
)

// terminalMessage WebSocket 终端消息协议
type terminalMessage struct {
	Type      string `json:"type"`
	Data      string `json:"data,omitempty"`
	Cols      int    `json:"cols,omitempty"`
	Rows      int    `json:"rows,omitempty"`
	Reason    string `json:"reason,omitempty"`
	SessionID string `json:"session_id,omitempty"`
}

// TerminalHandler WebSocket 终端处理器
type TerminalHandler struct {
	svc      *service.TerminalSessionService
	pool     *service.SSHPool
	jwt      *jwt.Manager
	logger   *zap.Logger
	upgrader websocket.Upgrader
}

// NewTerminalHandler 创建 TerminalHandler
func NewTerminalHandler(
	svc *service.TerminalSessionService,
	pool *service.SSHPool,
	jwt *jwt.Manager,
	allowedOrigins []string,
	logger *zap.Logger,
) *TerminalHandler {
	return &TerminalHandler{
		svc:    svc,
		pool:   pool,
		jwt:    jwt,
		logger: logger.Named("terminal"),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  4096,
			WriteBufferSize: 4096,
			CheckOrigin: func(r *http.Request) bool {
				return isWebSocketOriginAllowed(r, allowedOrigins)
			},
		},
	}
}

// HandleWebSocket 处理 WebSocket 终端连接
func (h *TerminalHandler) HandleWebSocket(c *gin.Context) {
	tokenString := webSocketToken(c)
	if tokenString == "" {
		response.Err(c, apperrors.New(apperrors.ErrUnauthorized, "缺少认证令牌"))
		c.Abort()
		return
	}
	claims, err := h.jwt.ValidateAccessToken(tokenString)
	if err != nil {
		h.logger.Debug("terminal ws auth failed", zap.Error(err))
		response.Err(c, apperrors.New(apperrors.ErrTokenInvalid, "令牌无效或已过期"))
		c.Abort()
		return
	}

	nodeID := c.Query("node_id")
	if nodeID == "" {
		response.Err(c, apperrors.New(apperrors.ErrBadRequest, "缺少 node_id 参数"))
		c.Abort()
		return
	}
	cols, _ := strconv.Atoi(c.DefaultQuery("cols", "80"))
	rows, _ := strconv.Atoi(c.DefaultQuery("rows", "24"))
	sessionID := c.Query("session_id")
	if cols < 40 {
		cols = 80
	}
	if rows < 10 {
		rows = 24
	}

	var sshClient *service.SSHClientEntry
	var sshSess *gossh.Session
	isNew := false

	if sessionID == "" {
		// ——— 创建新会话 ———
		sid, nodeEp, sshKey, sshPort, serr := h.svc.CreateSession(c.Request.Context(), claims.UserID, claims.Username, nodeID)
		if serr != nil {
			h.logger.Error("create session failed", zap.Error(serr))
			response.Err(c, apperrors.New(apperrors.ErrInternal, serr.Error()))
			c.Abort()
			return
		}
		sessionID = sid
		isNew = true

		// 建立 SSH 连接和 Session
		sshClient, err = h.pool.AcquireClient(c.Request.Context(), nodeID, nodeEp, sshPort, []byte(sshKey))
		if err != nil {
			h.svc.CloseSession(c.Request.Context(), sessionID, "error")
			h.logger.Error("ssh connect failed", zap.Error(err))
			response.Err(c, apperrors.New(apperrors.ErrInternal, "SSH 连接失败"))
			c.Abort()
			return
		}
		sshSess, err = h.pool.AcquireSession(c.Request.Context(), nodeID, sessionID, cols, rows)
		if err != nil {
			h.svc.CloseSession(c.Request.Context(), sessionID, "error")
			h.logger.Error("ssh session failed", zap.Error(err))
			response.Err(c, apperrors.New(apperrors.ErrInternal, "SSH 会话创建失败"))
			c.Abort()
			return
		}
	} else {
		// ——— 恢复会话 ———
		nodeEp, sshKey, sshPort, rerr := h.svc.ResumeSession(c.Request.Context(), sessionID, claims.UserID, nodeID)
		if rerr != nil {
			h.logger.Error("resume session failed", zap.Error(rerr))
			response.Err(c, apperrors.New(apperrors.ErrInternal, rerr.Error()))
			c.Abort()
			return
		}
		sshClient, err = h.pool.AcquireClient(c.Request.Context(), nodeID, nodeEp, sshPort, []byte(sshKey))
		if err != nil {
			h.svc.CloseSession(c.Request.Context(), sessionID, "error")
			h.logger.Error("ssh reconnect failed", zap.Error(err))
			response.Err(c, apperrors.New(apperrors.ErrInternal, "SSH 重连失败"))
			c.Abort()
			return
		}
		sshSess, err = h.pool.AcquireSession(c.Request.Context(), nodeID, sessionID, cols, rows)
		if err != nil {
			h.svc.CloseSession(c.Request.Context(), sessionID, "error")
			h.logger.Error("ssh session re-acquire failed", zap.Error(err))
			response.Err(c, apperrors.New(apperrors.ErrInternal, "SSH 会话重建失败"))
			c.Abort()
			return
		}
	}
	_ = sshClient

	// ——— 升级 WebSocket ———
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Error("websocket upgrade failed", zap.Error(err))
		sshSess.Close()
		if isNew {
			h.svc.CloseSession(c.Request.Context(), sessionID, "error")
		}
		return
	}

	// ——— 启动 Shell ———
	stdin, _ := sshSess.StdinPipe()
	stdout, _ := sshSess.StdoutPipe()
	stderr, _ := sshSess.StderrPipe()

	if err := sshSess.Shell(); err != nil {
		h.logger.Error("ssh shell start failed", zap.Error(err))
		conn.Close()
		sshSess.Close()
		if isNew {
			h.svc.CloseSession(c.Request.Context(), sessionID, "error")
		}
		return
	}

	// ——— I/O 桥接 ———
	h.bridgeIO(conn, stdin, stdout, stderr, sshSess, sessionID, claims.UserID, nodeID, cols, rows)

	h.logger.Info("terminal connected",
		zap.String("session_id", sessionID),
		zap.String("node_id", nodeID),
		zap.String("user_id", claims.UserID),
		zap.Bool("new", isNew),
	)
}

// bridgeIO 桥接 WebSocket ↔ SSH I/O
func (h *TerminalHandler) bridgeIO(conn *websocket.Conn, stdin io.WriteCloser, stdout, stderr io.Reader, sshSess *gossh.Session, sessionID, userID, nodeID string, cols, rows int) {
	var mu sync.Mutex
	closed := false

	writeJSON := func(msg terminalMessage) error {
		mu.Lock()
		defer mu.Unlock()
		if closed {
			return nil
		}
		return conn.WriteJSON(msg)
	}

	closeAll := func() {
		mu.Lock()
		closed = true
		mu.Unlock()
		conn.Close()
		sshSess.Close()
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = h.svc.PauseSession(ctx, sessionID, userID)
	}

	defer closeAll()

	var wg sync.WaitGroup
	wg.Add(3)

	// WebSocket ← 终端输出（stdout）
	go func() {
		defer wg.Done()
		buf := make([]byte, 4096)
		for {
			n, err := stdout.Read(buf)
			if err != nil {
				return
			}
			if n > 0 {
				_ = writeJSON(terminalMessage{Type: "output", Data: string(buf[:n])})
			}
		}
	}()

	// WebSocket ← 终端输出（stderr）
	go func() {
		defer wg.Done()
		buf := make([]byte, 4096)
		for {
			n, err := stderr.Read(buf)
			if err != nil {
				return
			}
			if n > 0 {
				_ = writeJSON(terminalMessage{Type: "output", Data: string(buf[:n])})
			}
		}
	}()

	// WebSocket → 终端输入（stdin + resize + ping）
	go func() {
		defer wg.Done()
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				return
			}
			var msg terminalMessage
			if err := json.Unmarshal(message, &msg); err != nil {
				continue
			}
			switch msg.Type {
			case "input":
				_, _ = stdin.Write([]byte(msg.Data))
			case "resize":
				if msg.Cols > 0 && msg.Rows > 0 {
					cols, rows = msg.Cols, msg.Rows
					_ = sshSess.WindowChange(msg.Rows, msg.Cols)
				}
			case "ping":
				_ = writeJSON(terminalMessage{Type: "pong"})
			}
		}
	}()

	wg.Wait()
}

// ListSessions 列出终端会话
func (h *TerminalHandler) ListSessions(c *gin.Context) {
	nodeID := c.Param("id")
	status := c.Query("status")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	sessions, total, err := h.svc.ListSessions(c.Request.Context(), nodeID, status, page, pageSize, "started_at", "desc")
	if err != nil {
		h.logger.Error("list sessions failed", zap.Error(err))
		response.Err(c, apperrors.New(apperrors.ErrInternal, "查询会话列表失败"))
		return
	}
	response.Page(c, sessions, total, page, pageSize)
}

// GetSession 获取会话详情
func (h *TerminalHandler) GetSession(c *gin.Context) {
	sessionID := c.Param("session_id")
	session, err := h.svc.GetSession(c.Request.Context(), sessionID)
	if err != nil {
		response.Err(c, apperrors.New(apperrors.ErrNotFound, "会话不存在"))
		return
	}
	response.OK(c, session)
}

// CloseSession 关闭终端会话
func (h *TerminalHandler) CloseSession(c *gin.Context) {
	sessionID := c.Param("session_id")
	if err := h.svc.ForceCloseSession(c.Request.Context(), sessionID); err != nil {
		h.logger.Error("close session failed", zap.Error(err))
		response.Err(c, apperrors.New(apperrors.ErrInternal, "关闭会话失败"))
		return
	}
	response.OK(c, nil)
}

// GetRecording 获取录制数据
func (h *TerminalHandler) GetRecording(c *gin.Context) {
	sessionID := c.Param("session_id")
	recording, err := h.svc.GetSessionRecording(c.Request.Context(), sessionID)
	if err != nil {
		response.Err(c, apperrors.New(apperrors.ErrNotFound, "录制数据不存在"))
		return
	}
	var data interface{}
	if len(recording) > 0 {
		_ = json.Unmarshal(recording, &data)
	}
	response.OK(c, data)
}
