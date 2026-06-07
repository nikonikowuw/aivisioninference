// Package ipc 提供 Go 控制面与 C++ 数据面之间的 IPC 通信支持。
// 包含 TCP 通信客户端/服务端实现，以及消息分发机制。

package ipc

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

// SignalType 信令类型，与 FlatBuffers envelope.fbs 定义一致
type SignalType uint16

const (
	// Go → C++ 指令 (0x0100 - 0x01FF)
	SignalStartStream        SignalType = 0x0100
	SignalStopStream         SignalType = 0x0101
	SignalUpdateAlgoConfig   SignalType = 0x0102
	SignalUpdateStreamConfig SignalType = 0x0103
	SignalHeartbeat          SignalType = 0x0104
	SignalStartSelfCheck     SignalType = 0x0105
	SignalShutdown           SignalType = 0x01FF

	// C++ → Go 结果 (0x0200 - 0x02FF)
	SignalInferenceResult   SignalType = 0x0200
	SignalStreamStatus      SignalType = 0x0201
	SignalAlgoLoadResult    SignalType = 0x0202
	SignalEngineMetrics     SignalType = 0x0203
	SignalHeartbeatAck      SignalType = 0x0204

	// 双向通用 (0x0300 - 0x03FF)
	SignalError             SignalType = 0x0300
)

// Message 表示一条原始 IPC 消息
type Message struct {
	SignalType SignalType
	Payload    []byte
	SequenceID uint64
	Timestamp  time.Time
}

// MessageHandler 消息处理器函数签名
type MessageHandler func(msg *Message)

// UDSReceiver TCP 消息接收器（兼容旧命名，内部已切换为 TCP）
// 负责：
//   1. 连接 C++ 引擎的 TCP Server
//   2. 接收原始字节流并解析为 Message
//   3. 按 SignalType 分发到注册的 Handler
//   4. 发送指令消息到 C++ 引擎
type UDSReceiver struct {
	addr     string
	conn     net.Conn
	mu         sync.RWMutex
	handlers   map[SignalType][]MessageHandler

	running    atomic.Bool
	sequenceID atomic.Uint64

	// 心跳监控
	lastHeartbeat time.Time
	hbMu          sync.RWMutex

	// 日志
	logger *zap.Logger
}

// UDSReceiverConfig 配置
type UDSReceiverConfig struct {
	Addr             string
	ReconnectDelay   time.Duration
	MaxReconnectWait time.Duration
}

// DefaultUDSReceiverConfig 默认配置
func DefaultUDSReceiverConfig(addr string) UDSReceiverConfig {
	return UDSReceiverConfig{
		Addr:             addr,
		ReconnectDelay:   1 * time.Second,
		MaxReconnectWait: 30 * time.Second,
	}
}

// NewUDSReceiver 创建 TCP 接收器
func NewUDSReceiver(cfg UDSReceiverConfig) *UDSReceiver {
	return &UDSReceiver{
		addr:     cfg.Addr,
		handlers: make(map[SignalType][]MessageHandler),
		logger:   zap.L().With(zap.String("component", "ipc_receiver")),
	}
}

// Connect 连接到 C++ 引擎 TCP Server
func (r *UDSReceiver) Connect() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.conn != nil {
		return nil
	}

	conn, err := net.DialTimeout("tcp", r.addr, 5*time.Second)
	if err != nil {
		return fmt.Errorf("connect to engine %s failed: %w", r.addr, err)
	}

	r.conn = conn
	r.running.Store(true)
	go r.readLoop()

	r.logger.Info("connected to C++ engine via TCP",
		zap.String("addr", r.addr))
	return nil
}

// Disconnect 断开连接
func (r *UDSReceiver) Disconnect() {
	r.running.Store(false)
	r.mu.Lock()
	if r.conn != nil {
		r.conn.Close()
		r.conn = nil
	}
	r.mu.Unlock()
}

// SendMessage 发送一条消息到 C++ 引擎
func (r *UDSReceiver) SendMessage(msg *Message) error {
	r.mu.RLock()
	conn := r.conn
	r.mu.RUnlock()

	if conn == nil {
		return fmt.Errorf("not connected to engine")
	}

	if msg.SequenceID == 0 {
		msg.SequenceID = r.sequenceID.Add(1)
	}

	// TODO: 使用 FlatBuffers IPCEnvelope 进行序列化
	// 当前使用简单的长度前缀 + SignalType 头部协议
	header := make([]byte, 10)
	binary.BigEndian.PutUint16(header[0:2], uint16(msg.SignalType))
	binary.BigEndian.PutUint64(header[2:10], msg.SequenceID)

	frame := append(header, msg.Payload...)
	if _, err := conn.Write(frame); err != nil {
		return fmt.Errorf("write to engine failed: %w", err)
	}

	return nil
}

// SendSignal 便捷方法：发送指定类型的空 payload 消息
func (r *UDSReceiver) SendSignal(signal SignalType) error {
	return r.SendMessage(&Message{
		SignalType: signal,
		Timestamp:  time.Now(),
	})
}

// RegisterHandler 注册消息处理器
func (r *UDSReceiver) RegisterHandler(signal SignalType, handler MessageHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[signal] = append(r.handlers[signal], handler)
}

// IsConnected 检查连接状态
func (r *UDSReceiver) IsConnected() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.conn != nil
}

// GetLastHeartbeat 获取最后心跳时间
func (r *UDSReceiver) GetLastHeartbeat() time.Time {
	r.hbMu.RLock()
	defer r.hbMu.RUnlock()
	return r.lastHeartbeat
}

// refreshHeartbeat 刷新心跳时间
func (r *UDSReceiver) refreshHeartbeat() {
	r.hbMu.Lock()
	r.lastHeartbeat = time.Now()
	r.hbMu.Unlock()
}

// readLoop 读取循环（goroutine）
func (r *UDSReceiver) readLoop() {
	defer func() {
		if err := recover(); err != nil {
			r.logger.Error("UDS read loop panicked", zap.Any("error", err))
		}
		r.Disconnect()
	}()

	for r.running.Load() {
		r.mu.RLock()
		conn := r.conn
		r.mu.RUnlock()

		if conn == nil {
			return
		}

		// 读取消息头 (8 字节: 4 SignalType LE + 4 PayloadLen LE)
		// 对齐 C++ IPCServer::SendResponse 线缆格式
		header := make([]byte, 8)
		if _, err := io.ReadFull(conn, header); err != nil {
			r.logger.Warn("UDS read header failed", zap.Error(err))
			r.reconnect()
			return
		}

		signalType := SignalType(binary.LittleEndian.Uint32(header[0:4]))
		payloadLen := binary.LittleEndian.Uint32(header[4:8])
		const maxPayloadSize = 16 * 1024 * 1024 // 16MB 上限
		if payloadLen > maxPayloadSize {
			r.logger.Error("UDS payload too large",
				zap.Uint32("payload_len", payloadLen))
			r.Disconnect()
			return
		}

		var payload []byte
		seqID := uint64(0) // 新格式无 sequence_id，默认 0
		if payloadLen > 0 {
			payload = make([]byte, payloadLen)
			if _, err := io.ReadFull(conn, payload); err != nil {
				r.logger.Warn("UDS read payload failed", zap.Error(err))
				r.reconnect()
				return
			}
		}

		msg := &Message{
			SignalType: signalType,
			Payload:    payload,
			SequenceID: seqID,
			Timestamp:  time.Now(),
		}

		// 心跳特殊处理
		if signalType == SignalHeartbeatAck {
			r.refreshHeartbeat()
		}

		// 分发消息
		r.dispatch(msg)
	}
}

// dispatch 分发消息到注册的 Handler
func (r *UDSReceiver) dispatch(msg *Message) {
	r.mu.RLock()
	handlers, ok := r.handlers[msg.SignalType]
	r.mu.RUnlock()

	if !ok {
		r.logger.Debug("no handler for signal type",
			zap.Uint16("signal_type", uint16(msg.SignalType)))
		return
	}

	for _, handler := range handlers {
		func(h MessageHandler, m *Message) {
			defer func() {
				if err := recover(); err != nil {
					r.logger.Error("handler panicked",
						zap.Uint16("signal_type", uint16(m.SignalType)),
						zap.Any("error", err))
				}
			}()
			h(m)
		}(handler, msg)
	}
}

// reconnect 重连逻辑
func (r *UDSReceiver) reconnect() {
	delay := 1 * time.Second
	for r.running.Load() {
		r.logger.Info("attempting to reconnect to C++ engine",
			zap.Duration("delay", delay))

		time.Sleep(delay)

		if err := r.Connect(); err == nil {
			r.logger.Info("reconnected to C++ engine")
			return
		}

		delay *= 2
		if delay > 30*time.Second {
			delay = 30 * time.Second
		}
	}
}

// MetricsReceiver 指标接收器
// 专门接收 C++ MetricsReporter 推送的 EngineMetricsMsg
type MetricsReceiver struct {
	receiver  *UDSReceiver
	onMetrics func(*EngineMetricsSnapshot)
	logger    *zap.Logger
}

// NewMetricsReceiver 创建指标接收器
func NewMetricsReceiver(receiver *UDSReceiver) *MetricsReceiver {
	mr := &MetricsReceiver{
		receiver: receiver,
		logger:   zap.L().With(zap.String("component", "metrics_receiver")),
	}

	// 注册指标消息处理器
	receiver.RegisterHandler(SignalEngineMetrics, mr.handleEngineMetrics)

	return mr
}

// SetOnMetrics 设置指标处理回调
func (mr *MetricsReceiver) SetOnMetrics(cb func(*EngineMetricsSnapshot)) {
	mr.onMetrics = cb
}

// handleEngineMetrics 处理引擎指标消息
func (mr *MetricsReceiver) handleEngineMetrics(msg *Message) {
	if len(msg.Payload) == 0 {
		return
	}

	// TODO: 使用 FlatBuffers 反序列化 EngineMetricsMsg
	// 当前使用 JSON 解析过渡
	snapshot := &EngineMetricsSnapshot{
		TimestampNS: uint64(msg.Timestamp.UnixNano()),
	}

	mr.logger.Debug("received engine metrics",
		zap.Uint64("timestamp", snapshot.TimestampNS))

	if mr.onMetrics != nil {
		mr.onMetrics(snapshot)
	}
}
