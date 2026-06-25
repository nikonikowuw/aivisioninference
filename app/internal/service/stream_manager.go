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

	// ZLM 外部可访问地址（引擎上报），供构造 HLS 播放 URL
	ZLMHost     string
	ZLMHTTPPort int
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

	// 锁内完成逻辑，保证并发安全
	state.mu.Lock()
	defer state.mu.Unlock()

	// 检查该 reason 是否已经存在，如果已存在则只刷新活跃时间，不增加引用计数
	if c, exists := state.Consumers.Load(reason); exists {
		consumer := c.(*ConsumerInfo)
		consumer.LastAlive = time.Now()
		// 如果是重复请求 play 且当前流正常，直接返回即可
		if state.Status == "active" || state.Status == "pulling" {
			return nil
		}
	} else {
		// 注册新的消费者
		consumer := &ConsumerInfo{
			Reason:    reason,
			RefAt:     time.Now(),
			Metadata:  metadata,
			LastAlive: time.Now(),
		}
		state.Consumers.Store(reason, consumer)
	}

	newCount := int32(0)
	state.Consumers.Range(func(key, value interface{}) bool {
		newCount++
		return true
	})
	state.RefCount.Store(newCount)

	if newCount == 1 {
		dev, err := m.deviceRepo.FindByID(ctx, deviceID)
		if err != nil {
			state.Consumers.Delete(reason)
			m.recalculateRefCount(state)
			return err
		}

		req := StreamStartRequest{
			DeviceID:       deviceID,
			RtspURL:        strings.TrimSpace(dev.RtspURL),
			EnableInfer:    reason == "infer",
			EnablePlayback: reason == "play",
		}
		if reason == "infer" && metadata != nil {
			req.AlgoName = metadata["algo_name"]
			req.AlgoVersion = metadata["algo_version"]
			req.SoPath = metadata["so_path"]
			req.AlgoParamsJSON = metadata["algo_params_json"]
		}

		info, err := m.engine.StartStream(ctx, req)
		if err != nil {
			state.Consumers.Delete(reason)
			m.recalculateRefCount(state)
			return err
		}

		state.Status = info.Status
		state.PlayURLRtsp = info.PlayURL
		state.ZLMHost = info.ZLMHost
		state.ZLMHTTPPort = info.ZLMHTTPPort

		go m.syncToDatabase(ctx, state, reason)
	} else {
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
				// 如果是由于之前状态残留导致 StartPlayback 失败，尝试完全重置状态并退回启动
				m.logger.Info("attempting to recover stream by starting over", zap.String("device_id", deviceID))
				info, retryErr := m.engine.StartStream(ctx, StreamStartRequest{
					DeviceID:       deviceID,
					RtspURL:        strings.TrimSpace(dev.RtspURL),
					EnableInfer:    false,
					EnablePlayback: true,
				})
				if retryErr != nil {
					return err // 返回原始错误
				}
				state.Status = info.Status
				state.PlayURLRtsp = info.PlayURL
			}
		}
	}

	return nil
}

func (m *StreamManager) recalculateRefCount(state *StreamState) {
	count := int32(0)
	state.Consumers.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	state.RefCount.Store(count)
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

	state.mu.Lock()
	defer state.mu.Unlock()

	state.Consumers.Delete(reason)
	m.recalculateRefCount(state)
	newCount := state.RefCount.Load()

	if newCount <= 0 {
		if err := m.engine.StopStream(ctx, deviceID); err != nil {
			m.logger.Error("stop engine stream failed", zap.Error(err), zap.String("device_id", deviceID))
		}
		state.Status = "inactive"

		go func() {
			dbCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			stream, err := m.streamRepo.FindByStream(dbCtx, "live", deviceID, "__defaultVhost__")
			if err == nil && stream != nil {
				_ = m.streamRepo.UpdateStatus(dbCtx, stream.ID, "inactive")
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

// ListStreams 列出所有活跃流状态（过滤掉无引用的 inactive 流）
func (m *StreamManager) ListStreams(ctx context.Context) []*StreamState {
	var results []*StreamState
	m.streams.Range(func(key, value interface{}) bool {
		state := value.(*StreamState)
		// 只返回有活跃引用或处于活跃状态的流
		if state.RefCount.Load() > 0 || state.Status == "active" || state.Status == "pulling" || state.Status == "error" {
			results = append(results, state)
		}
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
