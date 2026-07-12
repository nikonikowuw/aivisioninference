package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/niko-admin/niko-admin/internal/model"
	"github.com/niko-admin/niko-admin/internal/repository"
)

// TerminalSessionService 终端会话业务逻辑
type TerminalSessionService struct {
	sessionRepo *repository.TerminalSessionRepository
	nodeRepo    *repository.EdgeNodeRepository
	pool        *SSHPool
	logger      *zap.Logger

	maxSessionsPerNode int

	pausedMu    sync.Mutex
	pausedClean map[string]context.CancelFunc
}

// NewTerminalSessionService 创建 TerminalSessionService
func NewTerminalSessionService(
	sessionRepo *repository.TerminalSessionRepository,
	nodeRepo *repository.EdgeNodeRepository,
	pool *SSHPool,
	logger *zap.Logger,
) *TerminalSessionService {
	return &TerminalSessionService{
		sessionRepo:        sessionRepo,
		nodeRepo:           nodeRepo,
		pool:               pool,
		logger:             logger.Named("terminal_session"),
		maxSessionsPerNode: 10,
		pausedClean:        make(map[string]context.CancelFunc),
	}
}

// CreateSession 创建新终端会话：校验 → 写 DB → 返回连接参数
// 返回 (sessionID, nodeEndpoint, encryptedSSHKey, sshPort, error)
func (s *TerminalSessionService) CreateSession(ctx context.Context, userID, userName, nodeID string) (string, string, string, int, error) {
	count, err := s.sessionRepo.CountActiveByNodeID(ctx, nodeID)
	if err != nil {
		return "", "", "", 0, fmt.Errorf("count active sessions: %w", err)
	}
	if count >= int64(s.maxSessionsPerNode) {
		return "", "", "", 0, fmt.Errorf("节点 %s 已达最大并发会话数 (%d)", nodeID, s.maxSessionsPerNode)
	}

	node, err := s.nodeRepo.FindByID(ctx, nodeID)
	if err != nil {
		return "", "", "", 0, fmt.Errorf("查找节点失败: %w", err)
	}
	if node.SSHPrivateKey == "" {
		return "", "", "", 0, fmt.Errorf("节点 %s 未配置 SSH 私钥", nodeID)
	}
	if node.Status != model.NodeStatusOnline {
		return "", "", "", 0, fmt.Errorf("节点 %s 不在线 (状态: %s)", nodeID, node.Status)
	}

	sshPort := node.SSHPort
	if sshPort <= 0 {
		sshPort = 22
	}

	now := time.Now()
	session := &model.TerminalSession{
		NodeID:    nodeID,
		UserID:    userID,
		UserName:  userName,
		Status:    model.TerminalSessionStatusActive,
		StartedAt: now,
	}
	if err := s.sessionRepo.Create(ctx, session); err != nil {
		return "", "", "", 0, fmt.Errorf("保存会话记录失败: %w", err)
	}

	s.logger.Info("session created",
		zap.String("session_id", session.ID),
		zap.String("node_id", nodeID),
		zap.String("user_id", userID),
	)
	return session.ID, node.Endpoint, node.SSHPrivateKey, sshPort, nil
}

// ResumeSession 恢复暂停的会话并返回连接参数
func (s *TerminalSessionService) ResumeSession(ctx context.Context, sessionID, userID, nodeID string) (string, string, int, error) {
	session, err := s.sessionRepo.FindByID(ctx, sessionID)
	if err != nil {
		return "", "", 0, fmt.Errorf("会话不存在: %w", err)
	}
	if session.NodeID != nodeID {
		return "", "", 0, fmt.Errorf("会话 %s 不属于节点 %s", sessionID, nodeID)
	}
	if session.UserID != userID {
		return "", "", 0, fmt.Errorf("会话 %s 不属于当前用户", sessionID)
	}
	if session.Status != model.TerminalSessionStatusPaused {
		return "", "", 0, fmt.Errorf("会话 %s 状态不是 paused (当前: %s)", sessionID, session.Status)
	}

	s.pausedMu.Lock()
	if cancel, ok := s.pausedClean[sessionID]; ok {
		cancel()
		delete(s.pausedClean, sessionID)
	}
	s.pausedMu.Unlock()

	node, err := s.nodeRepo.FindByID(ctx, nodeID)
	if err != nil {
		return "", "", 0, fmt.Errorf("查找节点失败: %w", err)
	}

	sshPort := node.SSHPort
	if sshPort <= 0 {
		sshPort = 22
	}

	session.Status = model.TerminalSessionStatusActive
	session.PausedAt = nil
	if err := s.sessionRepo.Update(ctx, session); err != nil {
		return "", "", 0, fmt.Errorf("更新会话记录失败: %w", err)
	}

	s.logger.Info("session resumed",
		zap.String("session_id", sessionID),
		zap.String("node_id", nodeID),
	)
	return node.Endpoint, node.SSHPrivateKey, sshPort, nil
}

// PauseSession 暂停会话，启动 5 分钟超时清理
func (s *TerminalSessionService) PauseSession(ctx context.Context, sessionID, userID string) error {
	session, err := s.sessionRepo.FindByID(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("会话不存在: %w", err)
	}
	if session.UserID != userID {
		return fmt.Errorf("会话 %s 不属于当前用户", sessionID)
	}
	if session.Status != model.TerminalSessionStatusActive {
		return nil
	}

	now := time.Now()
	if err := s.sessionRepo.UpdateFields(ctx, sessionID, map[string]interface{}{
		"status":    model.TerminalSessionStatusPaused,
		"paused_at": now,
	}); err != nil {
		return fmt.Errorf("暂停会话失败: %w", err)
	}
	s.pool.ReleaseSession(session.NodeID, sessionID)
	s.pool.ReleaseClient(session.NodeID)

	cleanupCtx, cancel := context.WithCancel(context.Background())
	s.pausedMu.Lock()
	s.pausedClean[sessionID] = cancel
	s.pausedMu.Unlock()

	go func(sid, nid string) {
		select {
		case <-time.After(5 * time.Minute):
			_ = s.CloseSession(context.Background(), sid, "timeout")
		case <-cleanupCtx.Done():
		}
		cancel()
	}(sessionID, session.NodeID)

	s.logger.Info("session paused", zap.String("session_id", sessionID))
	return nil
}

// CloseSession 关闭会话
func (s *TerminalSessionService) CloseSession(ctx context.Context, sessionID, reason string) error {
	session, err := s.sessionRepo.FindByID(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("会话不存在: %w", err)
	}
	if session.Status == model.TerminalSessionStatusClosed {
		return nil
	}

	s.pausedMu.Lock()
	if cancel, ok := s.pausedClean[sessionID]; ok {
		cancel()
		delete(s.pausedClean, sessionID)
	}
	s.pausedMu.Unlock()

	s.pool.ReleaseSession(session.NodeID, sessionID)
	s.pool.ReleaseClient(session.NodeID)

	now := time.Now()
	duration := int(now.Sub(session.StartedAt).Seconds())
	if session.Status == model.TerminalSessionStatusPaused && session.PausedAt != nil {
		duration = int(session.PausedAt.Sub(session.StartedAt).Seconds())
	}
	if err := s.sessionRepo.UpdateFields(ctx, sessionID, map[string]interface{}{
		"status":           model.TerminalSessionStatusClosed,
		"reason":           reason,
		"ended_at":         now,
		"duration_seconds": duration,
	}); err != nil {
		return fmt.Errorf("关闭会话失败: %w", err)
	}

	s.logger.Info("session closed",
		zap.String("session_id", sessionID),
		zap.String("reason", reason),
	)
	return nil
}

// ListSessions 列出会话
func (s *TerminalSessionService) ListSessions(ctx context.Context, nodeID, status string, page, pageSize int, sort, order string) ([]model.TerminalSession, int64, error) {
	if sort == "" {
		sort = "started_at"
	}
	if order == "" {
		order = "desc"
	}
	return s.sessionRepo.ListByNodeID(ctx, nodeID, status, page, pageSize, sort, order)
}

// GetSession 获取会话详情
func (s *TerminalSessionService) GetSession(ctx context.Context, sessionID string) (*model.TerminalSession, error) {
	return s.sessionRepo.FindByID(ctx, sessionID)
}

// ForceCloseSession 强制关闭会话
func (s *TerminalSessionService) ForceCloseSession(ctx context.Context, sessionID string) error {
	return s.CloseSession(ctx, sessionID, "closed")
}

// GetSessionRecording 获取会话录制数据（TODO: 接入 Recorder）
func (s *TerminalSessionService) GetSessionRecording(ctx context.Context, sessionID string) ([]byte, error) {
	session, err := s.sessionRepo.FindByID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("会话不存在: %w", err)
	}
	return session.RecordingData, nil
}

// GetPool 返回 SSH 连接池（供 Handler 使用）
func (s *TerminalSessionService) GetPool() *SSHPool {
	return s.pool
}
