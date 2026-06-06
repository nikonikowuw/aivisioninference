package service

import (
	"testing"
	"time"
)

func TestHistoryBuffer_Add(t *testing.T) {
	buffer := NewHistoryBuffer(5)

	// 添加数据点
	now := time.Now()
	buffer.Add("cpu", 50.0, now)
	buffer.Add("cpu", 60.0, now.Add(time.Second))
	buffer.Add("cpu", 70.0, now.Add(2*time.Second))

	// 验证数据点数量
	points := buffer.Get("cpu", 5*time.Minute)
	if len(points) != 3 {
		t.Errorf("expected 3 points, got %d", len(points))
	}
}

func TestHistoryBuffer_MaxSize(t *testing.T) {
	buffer := NewHistoryBuffer(3)

	// 添加超过最大数量的数据点
	now := time.Now()
	for i := 0; i < 5; i++ {
		buffer.Add("cpu", float64(i), now.Add(time.Duration(i)*time.Second))
	}

	// 验证只保留最新的 3 个
	points := buffer.Get("cpu", 5*time.Minute)
	if len(points) != 3 {
		t.Errorf("expected 3 points, got %d", len(points))
	}

	// 验证是最新的 3 个
	if points[0].Value != 2.0 {
		t.Errorf("expected first point value 2.0, got %f", points[0].Value)
	}
}

func TestHistoryBuffer_GetDuration(t *testing.T) {
	buffer := NewHistoryBuffer(10)

	// 添加数据点
	now := time.Now()
	buffer.Add("cpu", 50.0, now.Add(-10*time.Minute))
	buffer.Add("cpu", 60.0, now.Add(-4*time.Minute))
	buffer.Add("cpu", 70.0, now)

	// 获取最近 5 分钟的数据
	points := buffer.Get("cpu", 5*time.Minute)
	if len(points) != 2 {
		t.Errorf("expected 2 points, got %d", len(points))
	}
}

func TestHistoryBuffer_GetNonExistentMetric(t *testing.T) {
	buffer := NewHistoryBuffer(10)

	points := buffer.Get("nonexistent", 5*time.Minute)
	if len(points) != 0 {
		t.Errorf("expected 0 points, got %d", len(points))
	}
}

func TestHistoryBuffer_Clear(t *testing.T) {
	buffer := NewHistoryBuffer(10)

	buffer.Add("cpu", 50.0, time.Now())
	buffer.Add("memory", 60.0, time.Now())

	buffer.Clear()

	points := buffer.Get("cpu", 5*time.Minute)
	if len(points) != 0 {
		t.Errorf("expected 0 points after clear, got %d", len(points))
	}

	points = buffer.Get("memory", 5*time.Minute)
	if len(points) != 0 {
		t.Errorf("expected 0 points after clear, got %d", len(points))
	}
}
