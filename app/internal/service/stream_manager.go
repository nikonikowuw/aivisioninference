package service

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"

	"github.com/niko-admin/niko-admin/internal/model"
)

// ConsumerInfo 记录流的消费者信息
type ConsumerInfo struct {
	Reason    string            `json:"reason"`
	RefAt     time.Time         `json:"ref_at"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	LastAlive time.Time         `json:"last_alive"`
}

// StreamState 描述流的内存运行时状态
type StreamState struct {
	DeviceID   string
	App        string
	Stream     string
	Vhost      string
	Schema     string
	RefCount   atomic.Int32
	mu         sync.Mutex // 保护 "检查-启动" 原子性，防止并发 Acquire 重复调用 engine
	Status     string     // inactive, pulling, active, error
	SourceURL  string
	RetryCount int
	RetryAt    *time.Time
	StartedAt  *time.Time
	Consumers  sync.Map // map[string]*ConsumerInfo, key=reason

	PlayURLRtsp   string
	PlayURLRtmp   string
	PlayURLFlv    string
	PlayURLWebrtc string
	PlayURLHls    string
}

// DeviceEvent 设备状态事件
type DeviceEvent struct {
	DeviceID  string
	EventType string // online, offline, error
	Timestamp time.Time
	Metadata  map[string]interface{}
}

// DeviceEventHandler 设备事件处理函数类型
type DeviceEventHandler func(ctx context.Context, event DeviceEvent) error

// streamRepo 接口抽象
type streamRepo interface {
	FindByStream(ctx context.Context, app, stream, vhost string) (*model.MediaStream, error)
	Create(ctx context.Context, item *model.MediaStream) error
	UpdateStatus(ctx context.Context, id, status string) error
}

// StreamManager 统一流管理器
type StreamManager struct {
	streams    sync.Map // map[string]*StreamState, key=deviceID
	engine     EngineClient
	deviceRepo deviceRepo
	streamRepo streamRepo
	logger     *zap.Logger

	handlers   []DeviceEventHandler
	muHandlers sync.RWMutex
}

// NewStreamManager 创建 StreamManager 实例
func NewStreamManager(engine EngineClient, devRepo deviceRepo, streamRepo streamRepo, logger *zap.Logger) *StreamManager {
	return &StreamManager{
		engine:     engine,
		deviceRepo: devRepo,
		streamRepo: streamRepo,
		logger:     logger,
	}
}

// Subscribe 订阅设备事件
func (m *StreamManager) Subscribe(handler DeviceEventHandler) {
	m.muHandlers.Lock()
	defer m.muHandlers.Unlock()
	m.handlers = append(m.handlers, handler)
}

// publish 发布事件
func (m *StreamManager) publish(ctx context.Context, event DeviceEvent) {
	m.muHandlers.RLock()
	handlers := m.handlers
	m.muHandlers.RUnlock()

	for _, h := range handlers {
		go func(handler DeviceEventHandler) {
			if err := handler(ctx, event); err != nil {
				m.logger.Error("event handler failed", zap.Error(err), zap.String("device_id", event.DeviceID))
			}
		}(h)
	}
}

// Acquire 申请引用一路流
func (m *StreamManager) Acquire(ctx context.Context, deviceID, reason string, metadata map[string]string) error {
	m.logger.Info("acquire stream", zap.String("device_id", deviceID), zap.String("reason", reason))

	actual, _ := m.streams.LoadOrStore(deviceID, &StreamState{
		DeviceID: deviceID,
		Status:   "inactive",
	})
	state := actual.(*StreamState)

	// 先注册消费者（在锁外，无需互斥）
	consumer := &ConsumerInfo{
		Reason:    reason,
		RefAt:     time.Now(),
		Metadata:  metadata,
		LastAlive: time.Now(),
	}
	state.Consumers.Store(reason, consumer)

	// 锁内完成 RefCount 判断 + engine 调用，防止并发 Acquire 重复启动流
	state.mu.Lock()
	newCount := state.RefCount.Add(1)

	if newCount == 1 {
		dev, err := m.deviceRepo.FindByID(ctx, deviceID)
		if err != nil {
			state.RefCount.Add(-1)
			state.Consumers.Delete(reason)
			state.mu.Unlock()
			return err
		}

		req := StreamStartRequest{
			DeviceID:       deviceID,
			RtspURL:        strings.TrimSpace(dev.RtspURL),
			EnableInfer:    reason == "infer",
			EnablePlayback: reason == "play",
		}

		info, err := m.engine.StartStream(ctx, req)
		if err != nil {
			state.RefCount.Add(-1)
			state.Consumers.Delete(reason)
			state.mu.Unlock()
			return err
		}

		state.Status = info.Status
		state.PlayURLRtsp = info.PlayURL
		state.mu.Unlock()

		go m.syncToDatabase(ctx, state, reason)
	} else {
		state.mu.Unlock()
		if reason == "play" {
			dev, err := m.deviceRepo.FindByID(ctx, deviceID)
			if err != nil {
				m.logger.Warn("load playback device failed", zap.Error(err), zap.String("device_id", deviceID))
				return err
			}
			_, err = m.engine.StartPlayback(ctx, StreamStartRequest{
				DeviceID:       deviceID,
				RtspURL:        dev.RtspURL,
				EnablePlayback: true,
			})
			if err != nil {
				m.logger.Warn("start playback failed", zap.Error(err), zap.String("device_id", deviceID))
				return err
			}
		}
	}

	return nil
}

func (m *StreamManager) syncToDatabase(ctx context.Context, state *StreamState, reason string) {
	item := &model.MediaStream{
		DeviceID:           state.DeviceID,
		ZLMApp:             "live",
		ZLMStream:          state.DeviceID,
		ZLMSchema:          "rtsp",
		PlayURLRtsp:        state.PlayURLRtsp,
		Status:             "active",
		ConsumerCount:      1,
		LastConsumerReason: reason,
	}
	_ = m.streamRepo.Create(ctx, item)
}

// Release 释放引用
func (m *StreamManager) Release(ctx context.Context, deviceID, reason string) error {
	m.logger.Info("release stream", zap.String("device_id", deviceID), zap.String("reason", reason))

	actual, ok := m.streams.Load(deviceID)
	if !ok {
		return nil
	}
	state := actual.(*StreamState)

	state.Consumers.Delete(reason)
	newCount := state.RefCount.Add(-1)

	if newCount <= 0 {
		if err := m.engine.StopStream(ctx, deviceID); err != nil {
			m.logger.Error("stop engine stream failed", zap.Error(err), zap.String("device_id", deviceID))
		}
		state.Status = "inactive"
		state.RefCount.Store(0)

		go func() {
			stream, err := m.streamRepo.FindByStream(ctx, "live", deviceID, "__defaultVhost__")
			if err == nil && stream != nil {
				_ = m.streamRepo.UpdateStatus(ctx, stream.ID, "inactive")
			}
		}()
	} else if reason == "play" {
		if err := m.engine.StopPlayback(ctx, deviceID); err != nil {
			m.logger.Warn("stop playback failed", zap.Error(err), zap.String("device_id", deviceID))
		}
	}

	return nil
}

// KeepAlive 更新消费者活跃时间
func (m *StreamManager) KeepAlive(ctx context.Context, deviceID, reason string) {
	actual, ok := m.streams.Load(deviceID)
	if !ok {
		return
	}
	state := actual.(*StreamState)
	if c, ok := state.Consumers.Load(reason); ok {
		consumer := c.(*ConsumerInfo)
		consumer.LastAlive = time.Now()
	}
}

// HandleStreamOnline 处理流上线事件
func (m *StreamManager) HandleStreamOnline(ctx context.Context, app, stream, vhost string) {
	m.logger.Info("stream online", zap.String("app", app), zap.String("stream", stream))

	deviceID := stream
	actual, ok := m.streams.Load(deviceID)
	if !ok {
		return
	}
	state := actual.(*StreamState)
	state.Status = "active"

	_ = m.deviceRepo.UpdateStatus(ctx, deviceID, "online", "", "")

	m.publish(ctx, DeviceEvent{
		DeviceID:  deviceID,
		EventType: "online",
		Timestamp: time.Now(),
	})
}

// HandleStreamOffline 处理流下线事件
func (m *StreamManager) HandleStreamOffline(ctx context.Context, app, stream, vhost string) {
	m.logger.Info("stream offline", zap.String("app", app), zap.String("stream", stream))

	deviceID := stream
	actual, ok := m.streams.Load(deviceID)
	if !ok {
		return
	}
	state := actual.(*StreamState)

	if state.RefCount.Load() > 0 {
		state.Status = "offline"
		_ = m.deviceRepo.UpdateStatus(ctx, deviceID, "offline", "", "stream disconnected unexpectedly")

		m.publish(ctx, DeviceEvent{
			DeviceID:  deviceID,
			EventType: "offline",
			Timestamp: time.Now(),
		})
	}
}

// HandleStreamNotFound 处理流找不到事件（重试逻辑）
func (m *StreamManager) HandleStreamNotFound(ctx context.Context, app, stream, vhost string) {
	deviceID := stream
	actual, ok := m.streams.Load(deviceID)
	if !ok {
		return
	}
	state := actual.(*StreamState)

	if state.RefCount.Load() <= 0 {
		return
	}

	if state.RetryCount >= 5 {
		m.logger.Warn("stream retry limit reached", zap.String("device_id", deviceID))
		state.Status = "error"
		m.publish(ctx, DeviceEvent{
			DeviceID:  deviceID,
			EventType: "error",
			Timestamp: time.Now(),
			Metadata:  map[string]interface{}{"error": "retry limit reached"},
		})
		return
	}

	state.RetryCount++
	intervals := []time.Duration{5, 30, 120, 300, 600}
	delay := intervals[state.RetryCount-1] * time.Second

	nextRetry := time.Now().Add(delay)
	state.RetryAt = &nextRetry

	m.logger.Info("scheduling stream retry",
		zap.String("device_id", deviceID),
		zap.Int("count", state.RetryCount),
		zap.Duration("delay", delay))

	go func() {
		time.Sleep(delay)
		m.reacquire(context.Background(), state)
	}()
}

func (m *StreamManager) reacquire(ctx context.Context, state *StreamState) {
	if state.RefCount.Load() <= 0 {
		return
	}

	dev, err := m.deviceRepo.FindByID(ctx, state.DeviceID)
	if err != nil {
		return
	}

	req := StreamStartRequest{
		DeviceID:       state.DeviceID,
		RtspURL:        dev.RtspURL,
		EnableInfer:    true,
		EnablePlayback: false,
	}

	_, _ = m.engine.StartStream(ctx, req)
}

// VerifyPlaybackAuth 验证播放权限
func (m *StreamManager) VerifyPlaybackAuth(ctx context.Context, app, stream, params string) error {
	return nil
}

// GetStream 获取单个流状态
func (m *StreamManager) GetStream(ctx context.Context, deviceID string) *StreamState {
	actual, ok := m.streams.Load(deviceID)
	if !ok {
		return nil
	}
	return actual.(*StreamState)
}

// ListStreams 列出所有流状态
func (m *StreamManager) ListStreams(ctx context.Context) []*StreamState {
	results := make([]*StreamState, 0)
	m.streams.Range(func(key, value interface{}) bool {
		results = append(results, value.(*StreamState))
		return true
	})
	return results
}

// StartBackgroundTasks 启动后台任务
func (m *StreamManager) StartBackgroundTasks(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// 使用独立 context，不受主 ctx 取消影响
				bgCtx := context.Background()
				m.checkConsumerTimeouts(bgCtx)
				m.syncWithEngine(bgCtx)
			}
		}
	}()
}

func (m *StreamManager) checkConsumerTimeouts(ctx context.Context) {
	now := time.Now()
	m.streams.Range(func(key, value interface{}) bool {
		state := value.(*StreamState)
		state.Consumers.Range(func(cKey, cValue interface{}) bool {
			consumer := cValue.(*ConsumerInfo)
			if now.Sub(consumer.LastAlive) > 5*time.Minute {
				m.logger.Warn("consumer timeout, releasing",
					zap.String("device_id", state.DeviceID),
					zap.String("reason", consumer.Reason))
				_ = m.Release(ctx, state.DeviceID, consumer.Reason)
			}
			return true
		})
		return true
	})
}

func (m *StreamManager) syncWithEngine(ctx context.Context) {
	m.streams.Range(func(key, value interface{}) bool {
		state := value.(*StreamState)
		if state.RefCount.Load() > 0 {
			info, err := m.engine.GetStreamStatus(ctx, state.DeviceID)
			if err != nil {
				m.logger.Warn("sync with engine failed", zap.Error(err), zap.String("device_id", state.DeviceID))
				return true
			}
			if info.Status == "inactive" {
				m.logger.Warn("stream lost in engine, reacquiring", zap.String("device_id", state.DeviceID))
				m.reacquire(ctx, state)
			}
		}
		return true
	})
}

// RecoverOnServerStart 重启后恢复流
func (m *StreamManager) RecoverOnServerStart(ctx context.Context) {
	m.logger.Info("recovering streams on server start")
	m.streams.Range(func(key, value interface{}) bool {
		state := value.(*StreamState)
		if state.RefCount.Load() > 0 {
			state.RetryCount = 0
			m.reacquire(ctx, state)
		}
		return true
	})
}

// GetStreamStatus 获取字符串形式的设备流状态（供外部调用）
func (m *StreamManager) GetStreamStatus(ctx context.Context, deviceID string) string {
	state := m.GetStream(ctx, deviceID)
	if state == nil {
		return "inactive"
	}
	return fmt.Sprintf("refcount=%d, status=%s", state.RefCount.Load(), state.Status)
}
