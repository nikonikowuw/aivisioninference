// Package service 提供业务逻辑层
package service

import (
	"sync"
	"time"
)

// HistoryPoint 历史数据点
type HistoryPoint struct {
	Time  time.Time `json:"time"`
	Value float64   `json:"value"`
}

// HistoryBuffer 环形缓冲区，用于存储历史数据
type HistoryBuffer struct {
	mu       sync.RWMutex
	data     map[string][]HistoryPoint
	maxSize  int
}

// NewHistoryBuffer 创建历史缓冲区
func NewHistoryBuffer(maxSize int) *HistoryBuffer {
	return &HistoryBuffer{
		data:    make(map[string][]HistoryPoint),
		maxSize: maxSize,
	}
}

// Add 添加数据点
func (b *HistoryBuffer) Add(metric string, value float64, t time.Time) {
	b.mu.Lock()
	defer b.mu.Unlock()

	points, ok := b.data[metric]
	if !ok {
		points = make([]HistoryPoint, 0, b.maxSize)
	}

	// 添加新数据点
	points = append(points, HistoryPoint{
		Time:  t,
		Value: value,
	})

	// 如果超过最大大小，移除最旧的数据点
	if len(points) > b.maxSize {
		points = points[len(points)-b.maxSize:]
	}

	b.data[metric] = points
}

// Get 获取指定时间范围内的数据点
func (b *HistoryBuffer) Get(metric string, duration time.Duration) []HistoryPoint {
	b.mu.RLock()
	defer b.mu.RUnlock()

	points, ok := b.data[metric]
	if !ok {
		return []HistoryPoint{}
	}

	cutoff := time.Now().Add(-duration)
	result := make([]HistoryPoint, 0, len(points))

	for _, p := range points {
		if p.Time.After(cutoff) {
			result = append(result, p)
		}
	}

	return result
}

// Clear 清空缓冲区
func (b *HistoryBuffer) Clear() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.data = make(map[string][]HistoryPoint)
}
