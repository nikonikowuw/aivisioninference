package service

import (
	"encoding/base64"
	"encoding/json"
	"sync"
	"time"

	"go.uber.org/zap"
)

// ttyrecEntry 表示一条 ttyrec 录制记录
type ttyrecEntry struct {
	Time float64 `json:"time"` // 会话内相对时间（秒）
	Data string  `json:"data"` // base64 编码的字节
}

// ttyRecorder 管理单个会话的 I/O 录制
type ttyRecorder struct {
	mu         sync.Mutex
	sessionID  string
	entries    []ttyrecEntry
	lastFlush  time.Time
	bufferSize int
	startTime  time.Time
	logger     *zap.Logger
}

const (
	recorderFlushInterval = 5 * time.Second
	recorderFlushBatch    = 100
	recorderMaxEntries    = 10000 // 约 10MB 阈值保护
)

// Recorder 管理所有活跃终端会话的 I/O 录制
type Recorder struct {
	sessions sync.Map // sessionID → *ttyRecorder
	logger   *zap.Logger
	flushCh  chan string // sessionID 通知 flush
}

// NewRecorder 创建 Recorder 实例
func NewRecorder(logger *zap.Logger) *Recorder {
	return &Recorder{
		logger:  logger.Named("terminal_recorder"),
		flushCh: make(chan string, 256),
	}
}

// StartRecording 开始录制指定会话的 I/O
func (r *Recorder) StartRecording(sessionID string) {
	rec := &ttyRecorder{
		sessionID: sessionID,
		entries:   make([]ttyrecEntry, 0, 1024),
		lastFlush: time.Now(),
		startTime: time.Now(),
		logger:    r.logger,
	}
	r.sessions.Store(sessionID, rec)

	r.logger.Debug("recording started", zap.String("session_id", sessionID))
}

// Record 写入 I/O 数据到录制 buffer（非阻塞）
func (r *Recorder) Record(sessionID string, data []byte) {
	val, ok := r.sessions.Load(sessionID)
	if !ok {
		return
	}
	rec := val.(*ttyRecorder)

	rec.mu.Lock()
	defer rec.mu.Unlock()

	// 超过上限则丢弃并记录警告
	if len(rec.entries) >= recorderMaxEntries {
		rec.logger.Warn("recording buffer full, dropping data",
			zap.String("session_id", sessionID),
		)
		return
	}

	elapsed := time.Since(rec.startTime).Seconds()
	entry := ttyrecEntry{
		Time: elapsed,
		Data: base64.StdEncoding.EncodeToString(data),
	}
	rec.entries = append(rec.entries, entry)
	rec.bufferSize++

	// 达到批量阈值时触发 flush 通知
	if rec.bufferSize >= recorderFlushBatch {
		select {
		case r.flushCh <- sessionID:
		default:
			// 通知队列满，跳过（下次自动 flush）
		}
	}
}

// Flush 强制将指定会话的 buffer 写入并返回 JSON 序列化后的批次
func (r *Recorder) Flush(sessionID string) ([][]byte, bool) {
	val, ok := r.sessions.Load(sessionID)
	if !ok {
		return nil, false
	}
	rec := val.(*ttyRecorder)

	rec.mu.Lock()
	defer rec.mu.Unlock()

	if len(rec.entries) == 0 {
		return nil, false
	}

	entries := rec.entries
	rec.entries = make([]ttyrecEntry, 0, 1024)
	rec.bufferSize = 0
	rec.lastFlush = time.Now()

	// 按批次切分（每次 Asynq 消息不超过 1MB）
	const maxBatchSize = 800
	var batches [][]byte
	for i := 0; i < len(entries); i += maxBatchSize {
		end := i + maxBatchSize
		if end > len(entries) {
			end = len(entries)
		}
		data, err := json.Marshal(entries[i:end])
		if err != nil {
			r.logger.Warn("failed to marshal recording batch", zap.Error(err))
			continue
		}
		batches = append(batches, data)
	}

	return batches, true
}

// StopRecording 停止录制并返回 accumulated 录制数据（JSON 序列化后的 ttyrec 数组）
func (r *Recorder) StopRecording(sessionID string) ([]byte, bool) {
	val, ok := r.sessions.LoadAndDelete(sessionID)
	if !ok {
		return nil, false
	}
	rec := val.(*ttyRecorder)

	rec.mu.Lock()
	defer rec.mu.Unlock()

	if len(rec.entries) == 0 {
		return nil, false
	}

	data, err := json.Marshal(rec.entries)
	if err != nil {
		r.logger.Warn("failed to marshal recording entries", zap.Error(err))
		return nil, false
	}
	r.logger.Debug("recording stopped",
		zap.String("session_id", sessionID),
		zap.Int("total_entries", len(rec.entries)),
	)
	return data, true
}

// FlushChannel 返回 flush 通知 channel（供外部 goroutine 消费）
func (r *Recorder) FlushChannel() <-chan string {
	return r.flushCh
}
