// Package service 提供业务逻辑层
package service

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/niko-admin/niko-admin/internal/model"
)

// BatchWriterConfig 批量写入器配置
type BatchWriterConfig struct {
	// 批处理缓冲区大小（达到此数量触发写入）
	BatchSize int
	// 最大等待时间（达到此时间即使未满也触发写入）
	FlushInterval time.Duration
	// 最大并行写入goroutine数
	MaxConcurrent int
}

// DefaultBatchWriterConfig 返回默认配置
func DefaultBatchWriterConfig() BatchWriterConfig {
	return BatchWriterConfig{
		BatchSize:     100,
		FlushInterval: 500 * time.Millisecond,
		MaxConcurrent: 2,
	}
}

// BatchWriter 基于 Channel 与 Ticker 的内存批量写入器
// 用于高频 SmartRecord 批量入库，保护 PostgreSQL 不被高频单条 INSERT 锁死。
type BatchWriter struct {
	ctx       context.Context
	cancel    context.CancelFunc
	db        *gorm.DB
	config    BatchWriterConfig
	recordsCh chan *model.SmartRecord
	wg        sync.WaitGroup
	logger    *zap.Logger
	flushSem  chan struct{} // 控制并发写入 goroutine 数量
}

// NewBatchWriter 创建批量写入器
func NewBatchWriter(db *gorm.DB, config BatchWriterConfig) *BatchWriter {
	ctx, cancel := context.WithCancel(context.Background())
	maxConcurrent := config.MaxConcurrent
	if maxConcurrent <= 0 {
		maxConcurrent = 2
	}
	return &BatchWriter{
		ctx:       ctx,
		cancel:    cancel,
		db:        db,
		config:    config,
		recordsCh: make(chan *model.SmartRecord, config.BatchSize*10),
		logger:    zap.L().With(zap.String("component", "batch_writer")),
		flushSem:  make(chan struct{}, maxConcurrent),
	}
}

// Start 启动批量写入器
func (w *BatchWriter) Start() {
	w.logger.Info("starting batch writer",
		zap.Int("batch_size", w.config.BatchSize),
		zap.Duration("flush_interval", w.config.FlushInterval))

	w.wg.Add(1)
	go w.processLoop()
}

// Stop 优雅停止批量写入器
func (w *BatchWriter) Stop() {
	w.logger.Info("stopping batch writer")
	w.cancel()
	w.wg.Wait()
	w.logger.Info("batch writer stopped")
}

// Submit 提交一条记录到写入队列
func (w *BatchWriter) Submit(record *model.SmartRecord) {
	select {
	case w.recordsCh <- record:
	default:
		w.logger.Warn("batch writer buffer full, dropping record",
			zap.String("record_type", record.RecordType))
	}
}

// processLoop 处理主循环
func (w *BatchWriter) processLoop() {
	defer w.wg.Done()

	batch := make([]*model.SmartRecord, 0, w.config.BatchSize)
	ticker := time.NewTicker(w.config.FlushInterval)
	defer ticker.Stop()

	flush := func() {
		if len(batch) == 0 {
			return
		}

		// 深拷贝当前批次并重置
		toFlush := make([]*model.SmartRecord, len(batch))
		copy(toFlush, batch)
		batch = batch[:0]

		// 并发控制：通过信号量限制同时运行的写入 goroutine 数
		w.wg.Add(1)
		go func(records []*model.SmartRecord) {
			defer w.wg.Done()
			w.flushSem <- struct{}{}
			defer func() { <-w.flushSem }()
			w.flushToDB(records)
		}(toFlush)
	}

	for {
		select {
		case <-w.ctx.Done():
			// 退出前先刷新当前 batch
			flush()
			// drain channel 中剩余记录
			for {
				select {
				case record := <-w.recordsCh:
					batch = append(batch, record)
					if len(batch) >= w.config.BatchSize {
						flush()
					}
				default:
					flush()
					return
				}
			}

		case record := <-w.recordsCh:
			batch = append(batch, record)
			if len(batch) >= w.config.BatchSize {
				flush()
			}

		case <-ticker.C:
			flush()
		}
	}
}

// flushToDB 将一批记录写入数据库
func (w *BatchWriter) flushToDB(records []*model.SmartRecord) {
	if len(records) == 0 {
		return
	}

	start := time.Now()

	// 使用独立的带超时 context，不依赖已取消的 w.ctx
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	values := make([]interface{}, len(records))
	for i, r := range records {
		values[i] = r
	}

	if err := w.db.WithContext(ctx).CreateInBatches(values, len(records)).Error; err != nil {
		w.logger.Error("batch write failed",
			zap.Int("count", len(records)),
			zap.Error(err))
		return
	}

	elapsed := time.Since(start)
	w.logger.Debug("batch write completed",
		zap.Int("count", len(records)),
		zap.Duration("elapsed", elapsed))
}
