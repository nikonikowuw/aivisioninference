package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/niko-admin/niko-admin/internal/dto"
)

// TerminalSession tracks an active Web Terminal session.
type TerminalSession struct {
	SessionID     string
	NodeID        string
	LastActivity  time.Time
	IdleTimeout   time.Duration
	Done          chan struct{}
	WriteCallback func(data []byte) // Called to send data to WS
}

// EdgeNodeTerminalService manages WebSocket-to-MQTT PTY proxy sessions.
type EdgeNodeTerminalService struct {
	mqttClient    mqtt.Client
	sessions      sync.Map // sessionID -> *TerminalSession
	idleTimeout   time.Duration
	stopCleanup   chan struct{}
}

// NewEdgeNodeTerminalService creates a new terminal service.
func NewEdgeNodeTerminalService(
	mqttClient mqtt.Client,
) *EdgeNodeTerminalService {
	svc := &EdgeNodeTerminalService{
		mqttClient:  mqttClient,
		idleTimeout: 5 * time.Minute, // Default 5 min idle timeout
		stopCleanup: make(chan struct{}),
	}

	go svc.cleanupLoop()
	return svc
}

// SetIdleTimeout sets the idle timeout for terminal sessions.
func (s *EdgeNodeTerminalService) SetIdleTimeout(timeout time.Duration) {
	if timeout > 0 {
		s.idleTimeout = timeout
	}
}

// OpenSession opens a new PTY session on the edge node.
// Returns the session ID or an error.
func (s *EdgeNodeTerminalService) OpenSession(ctx context.Context, nodeID string, writeCallback func(data []byte)) (string, error) {
	if s.mqttClient == nil {
		return "", fmt.Errorf("MQTT client not available")
	}

	sessionID := uuid.New().String()
	traceID := uuid.New().String()

	topic := fmt.Sprintf("aivision/edge/%s/cmd/pty_open", nodeID)
	payload := map[string]interface{}{
		"trace_id":   traceID,
		"session_id": sessionID,
	}
	payloadBytes, _ := json.Marshal(payload)

	token := s.mqttClient.Publish(topic, 1, false, payloadBytes)
	token.Wait()
	if token.Error() != nil {
		return "", fmt.Errorf("MQTT pty_open publish failed: %w", token.Error())
	}

	session := &TerminalSession{
		SessionID:     sessionID,
		NodeID:        nodeID,
		LastActivity:  time.Now(),
		IdleTimeout:   s.idleTimeout,
		Done:          make(chan struct{}),
		WriteCallback: writeCallback,
	}
	s.sessions.Store(sessionID, session)

	zap.L().Info("terminal session opened",
		zap.String("session_id", sessionID),
		zap.String("node_id", nodeID),
	)

	return sessionID, nil
}

// WriteToSession sends terminal input to the edge node PTY via MQTT.
func (s *EdgeNodeTerminalService) WriteToSession(ctx context.Context, sessionID string, data string) error {
	if s.mqttClient == nil {
		return fmt.Errorf("MQTT client not available")
	}

	sessionRaw, ok := s.sessions.Load(sessionID)
	if !ok {
		return fmt.Errorf("session not found: %s", sessionID)
	}
	session := sessionRaw.(*TerminalSession)
	session.LastActivity = time.Now()

	traceID := uuid.New().String()
	topic := fmt.Sprintf("aivision/edge/%s/cmd/pty_write", session.NodeID)
	payload := map[string]interface{}{
		"trace_id":   traceID,
		"session_id": sessionID,
		"data":       data,
	}
	payloadBytes, _ := json.Marshal(payload)

	token := s.mqttClient.Publish(topic, 1, false, payloadBytes)
	token.Wait()
	return token.Error()
}

// ResizeSession sends terminal resize event to the edge node PTY via MQTT.
func (s *EdgeNodeTerminalService) ResizeSession(ctx context.Context, sessionID string, cols, rows int) error {
	if s.mqttClient == nil {
		return fmt.Errorf("MQTT client not available")
	}

	sessionRaw, ok := s.sessions.Load(sessionID)
	if !ok {
		return fmt.Errorf("session not found: %s", sessionID)
	}
	session := sessionRaw.(*TerminalSession)

	traceID := uuid.New().String()
	topic := fmt.Sprintf("aivision/edge/%s/cmd/pty_resize", session.NodeID)
	payload := map[string]interface{}{
		"trace_id":   traceID,
		"session_id": sessionID,
		"cols":       cols,
		"rows":       rows,
	}
	payloadBytes, _ := json.Marshal(payload)

	token := s.mqttClient.Publish(topic, 1, false, payloadBytes)
	token.Wait()
	return token.Error()
}

// CloseSession closes a terminal session and sends pty_close to the edge node.
func (s *EdgeNodeTerminalService) CloseSession(ctx context.Context, sessionID string) error {
	sessionRaw, ok := s.sessions.Load(sessionID)
	if !ok {
		return nil // Already closed
	}
	session := sessionRaw.(*TerminalSession)

	// Mark as done to stop cleanup
	select {
	case <-session.Done:
		// Already closed
	default:
		close(session.Done)
	}

	s.sessions.Delete(sessionID)

	if s.mqttClient != nil {
		traceID := uuid.New().String()
		topic := fmt.Sprintf("aivision/edge/%s/cmd/pty_close", session.NodeID)
		payload := map[string]interface{}{
			"trace_id":   traceID,
			"session_id": sessionID,
		}
		payloadBytes, _ := json.Marshal(payload)

		token := s.mqttClient.Publish(topic, 1, false, payloadBytes)
		token.Wait()
		if token.Error() != nil {
			zap.L().Error("failed to publish pty_close",
				zap.String("session_id", sessionID),
				zap.Error(token.Error()),
			)
		}
	}

	zap.L().Info("terminal session closed",
		zap.String("session_id", sessionID),
		zap.String("node_id", session.NodeID),
	)

	return nil
}

// HandlePtyOutput handles incoming PTY output from MQTT and forwards to the WS client.
func (s *EdgeNodeTerminalService) HandlePtyOutput(nodeID string, payload map[string]interface{}) {
	sessionID, _ := payload["session_id"].(string)
	data, _ := payload["data"].(string)

	if sessionID == "" {
		return
	}

	sessionRaw, ok := s.sessions.Load(sessionID)
	if !ok {
		zap.L().Warn("pty_output for unknown session",
			zap.String("session_id", sessionID),
			zap.String("node_id", nodeID),
		)
		return
	}
	session := sessionRaw.(*TerminalSession)
	session.LastActivity = time.Now()

	if session.WriteCallback != nil && data != "" {
		msg := dto.WebSocketTerminalMessage{
			Type:      "output",
			SessionID: sessionID,
			Data:      data,
		}
		msgBytes, _ := json.Marshal(msg)
		session.WriteCallback(msgBytes)
	}
}

// HandlePtyError handles incoming PTY error from MQTT.
func (s *EdgeNodeTerminalService) HandlePtyError(nodeID string, payload map[string]interface{}) {
	sessionID, _ := payload["session_id"].(string)
	errorMsg, _ := payload["error"].(string)

	if sessionID == "" {
		return
	}

	sessionRaw, ok := s.sessions.Load(sessionID)
	if !ok {
		return
	}
	session := sessionRaw.(*TerminalSession)

	// Forward error to WS client
	if session.WriteCallback != nil {
		msg := dto.WebSocketTerminalMessage{
			Type:      "error",
			SessionID: sessionID,
			Error:     errorMsg,
		}
		msgBytes, _ := json.Marshal(msg)
		session.WriteCallback(msgBytes)
	}

	// Close session
	_ = s.CloseSession(context.Background(), sessionID)
}

// cleanupLoop periodically checks for idle sessions and closes them.
func (s *EdgeNodeTerminalService) cleanupLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.cleanupIdleSessions()
		case <-s.stopCleanup:
			return
		}
	}
}

// cleanupIdleSessions closes sessions that have exceeded the idle timeout.
func (s *EdgeNodeTerminalService) cleanupIdleSessions() {
	now := time.Now()
	s.sessions.Range(func(key, value interface{}) bool {
		session := value.(*TerminalSession)
		if now.Sub(session.LastActivity) > session.IdleTimeout {
			zap.L().Info("closing idle terminal session",
				zap.String("session_id", session.SessionID),
				zap.String("node_id", session.NodeID),
				zap.Duration("idle_duration", now.Sub(session.LastActivity)),
			)
			_ = s.CloseSession(context.Background(), session.SessionID)
		}
		return true
	})
}

// Stop gracefully stops the cleanup loop.
func (s *EdgeNodeTerminalService) Stop() {
	select {
	case <-s.stopCleanup:
	default:
		close(s.stopCleanup)
	}

	// Close all sessions
	s.sessions.Range(func(key, value interface{}) bool {
		session := value.(*TerminalSession)
		_ = s.CloseSession(context.Background(), session.SessionID)
		return true
	})
}
