// Package service provides business logic layer.
package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/niko-admin/niko-admin/internal/pkg/cryptoutil"
	gossh "golang.org/x/crypto/ssh"
	"go.uber.org/zap"
)

// SSHClientEntry 封装一个 ssh.Client 及其引用的 SSH Session 集合
type SSHClientEntry struct {
	Client      *gossh.Client
	NodeID      string
	ConnectedAt time.Time

	mu       sync.Mutex
	sessions map[string]*gossh.Session // key = sessChKey
}

// Close 关闭 SSH Client 并清理所有活跃 Session
func (e *SSHClientEntry) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	for key, sess := range e.sessions {
		sess.Close()
		delete(e.sessions, key)
	}
	return e.Client.Close()
}

// SSHPool 管理按 nodeID 池化的 SSH 连接
type SSHPool struct {
	mu      sync.RWMutex
	clients map[string]*SSHPoolEntry // key = nodeID
	logger  *zap.Logger

	maxIdleDuration time.Duration
}

type SSHPoolEntry struct {
	client    *SSHClientEntry
	refCount  int
	idleTimer *time.Timer
	closed    bool
}

// NewSSHPool 创建 SSH 连接池
func NewSSHPool() *SSHPool {
	return &SSHPool{
		clients:         make(map[string]*SSHPoolEntry),
		logger:          zap.L().Named("ssh_pool"),
		maxIdleDuration: 5 * time.Minute,
	}
}

// AcquireClient 获取或创建指向指定节点的 SSH Client
// encryptedKey 为 AES-256-GCM 加密后的 SSH 私钥
func (p *SSHPool) AcquireClient(ctx context.Context, nodeID, endpoint string, sshPort int, encryptedKey []byte) (*SSHClientEntry, error) {
	if encryptedKey == nil {
		return nil, fmt.Errorf("SSH private key is not configured for node %s", nodeID)
	}

	p.mu.Lock()

	// 已有池化条目，引用计数+1
	if entry, ok := p.clients[nodeID]; ok && !entry.closed {
		if entry.idleTimer != nil {
			entry.idleTimer.Stop()
			entry.idleTimer = nil
		}
		entry.refCount++
		p.mu.Unlock()
		return entry.client, nil
	}
	p.mu.Unlock()

	// 解密 SSH 私钥
	decryptedKey, err := decryptSSHKey(encryptedKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt SSH key for node %s: %w", nodeID, err)
	}

	signer, err := gossh.ParsePrivateKey(decryptedKey)
	if err != nil {
		return nil, fmt.Errorf("failed to parse SSH private key for node %s: %w", nodeID, err)
	}

	config := &gossh.ClientConfig{
		User:            "admin",
		Auth:            []gossh.AuthMethod{gossh.PublicKeys(signer)},
		HostKeyCallback: gossh.InsecureIgnoreHostKey(), // 内网设备，忽略 host key 校验
		Timeout:         10 * time.Second,
	}

	addr := fmt.Sprintf("%s:%d", endpoint, sshPort)
	client, err := gossh.Dial("tcp", addr, config)
	if err != nil {
		return nil, fmt.Errorf("failed to SSH dial node %s (%s): %w", nodeID, addr, err)
	}

	sshClientEntry := &SSHClientEntry{
		Client:      client,
		NodeID:      nodeID,
		ConnectedAt: time.Now(),
		sessions:    make(map[string]*gossh.Session),
	}

	p.mu.Lock()
	poolEntry := &SSHPoolEntry{
		client:   sshClientEntry,
		refCount: 1,
	}
	p.clients[nodeID] = poolEntry
	p.mu.Unlock()

	p.logger.Info("SSH connection established",
		zap.String("node_id", nodeID),
		zap.String("addr", addr),
	)

	return sshClientEntry, nil
}

// ReleaseClient 释放对某节点 SSH Client 的引用
// 引用计数归零后会启动 idle 超时定时器，超时后自动关闭连接
func (p *SSHPool) ReleaseClient(nodeID string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	entry, ok := p.clients[nodeID]
	if !ok || entry.closed {
		return
	}

	entry.refCount--
	if entry.refCount < 0 {
		entry.refCount = 0
	}

	if entry.refCount == 0 {
		// 启动 5 分钟 idle 超时
		entry.idleTimer = time.AfterFunc(p.maxIdleDuration, func() {
			p.closeClient(nodeID)
		})
	}
}

// AcquireSession 在已有 SSH Client 上创建新的 SSH Session
func (p *SSHPool) AcquireSession(ctx context.Context, nodeID, sessChKey string, cols, rows int) (*gossh.Session, error) {
	p.mu.RLock()
	entry, ok := p.clients[nodeID]
	p.mu.RUnlock()

	if !ok || entry.closed || entry.client == nil {
		return nil, fmt.Errorf("SSH client for node %s is not connected", nodeID)
	}

	clientEntry := entry.client

	session, err := clientEntry.Client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create SSH session for node %s: %w", nodeID, err)
	}

	// 请求 PTY
	modes := gossh.TerminalModes{
		gossh.ECHO:          1,
		gossh.TTY_OP_ISPEED: 14400,
		gossh.TTY_OP_OSPEED: 14400,
	}
	if err := session.RequestPty("xterm-256color", rows, cols, modes); err != nil {
		session.Close()
		return nil, fmt.Errorf("failed to request PTY for node %s: %w", nodeID, err)
	}

	clientEntry.mu.Lock()
	clientEntry.sessions[sessChKey] = session
	clientEntry.mu.Unlock()

	p.logger.Debug("SSH session acquired",
		zap.String("node_id", nodeID),
		zap.String("sess_ch_key", sessChKey),
	)

	return session, nil
}

// ReleaseSession 释放并关闭指定的 SSH Session
func (p *SSHPool) ReleaseSession(nodeID, sessChKey string) {
	p.mu.RLock()
	entry, ok := p.clients[nodeID]
	p.mu.RUnlock()

	if !ok || entry.closed {
		return
	}

	clientEntry := entry.client
	clientEntry.mu.Lock()
	if sess, exists := clientEntry.sessions[sessChKey]; exists {
		sess.Close()
		delete(clientEntry.sessions, sessChKey)
	}
	clientEntry.mu.Unlock()
}

// Keepalive 向所有活跃 SSH Client 发送保活请求
func (p *SSHPool) Keepalive(ctx context.Context) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for nodeID, entry := range p.clients {
		if entry.closed || entry.client == nil {
			continue
		}
		_, _, err := entry.client.Client.SendRequest("keepalive@openssh.com", true, nil)
		if err != nil {
			p.logger.Warn("SSH keepalive failed",
				zap.String("node_id", nodeID),
				zap.Error(err),
			)
		}
	}
}

// CloseAll 关闭所有 SSH 连接（应用退出时调用）
func (p *SSHPool) CloseAll() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for nodeID, entry := range p.clients {
		if entry.idleTimer != nil {
			entry.idleTimer.Stop()
		}
		if !entry.closed {
			entry.closed = true
			entry.client.Close()
		}
		delete(p.clients, nodeID)
	}
}

// closeClient 内部关闭指定节点的 SSH Client
func (p *SSHPool) closeClient(nodeID string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	entry, ok := p.clients[nodeID]
	if !ok || entry.closed {
		return
	}

	if entry.refCount > 0 {
		return // 又被引用了，跳过关闭
	}

	entry.closed = true
	entry.client.Close()
	delete(p.clients, nodeID)

	p.logger.Info("SSH connection closed due to idle timeout",
		zap.String("node_id", nodeID),
	)
}

// decryptSSHKey 解密 SSH 私钥
func decryptSSHKey(encrypted []byte) ([]byte, error) {
	return cryptoutil.DecryptSSHKey(encrypted)
}
