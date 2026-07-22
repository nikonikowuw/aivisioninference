package service

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGeneratePlayToken(t *testing.T) {
	svc := &MediaService{
		zlmSecret: "test-secret",
	}

	token := svc.GeneratePlayToken("stream-123", "user-456", 3600)
	assert.Contains(t, token, "user-456")
	assert.Contains(t, token, "_")
}

func TestStreamViewerTracker_AddRemove(t *testing.T) {
	tracker := NewStreamViewerTracker()

	// Add viewers
	assert.Equal(t, 1, tracker.AddViewer("live/test"))
	assert.Equal(t, 2, tracker.AddViewer("live/test"))
	assert.Equal(t, 1, tracker.AddViewer("live/other"))

	assert.True(t, tracker.HasViewers("live/test"))
	assert.Equal(t, 2, tracker.GetViewerCount("live/test"))

	// Remove viewers
	assert.Equal(t, 1, tracker.RemoveViewer("live/test"))
	assert.Equal(t, 0, tracker.RemoveViewer("live/test"))
	assert.False(t, tracker.HasViewers("live/test"))

	// Remove non-existent stream
	assert.Equal(t, 0, tracker.RemoveViewer("nonexistent"))
}

func TestStreamViewerTracker_RemoveZero(t *testing.T) {
	tracker := NewStreamViewerTracker()
	// Remove from empty - should not panic
	assert.Equal(t, 0, tracker.RemoveViewer("empty-stream"))
}

func TestNewMediaService(t *testing.T) {
	svc := NewMediaService(nil, nil, nil, nil, nil, "http://localhost:8000", "secret")
	assert.NotNil(t, svc)
	assert.Equal(t, "secret", svc.zlmSecret)
}

// TestStreamViewerTracker_Concurrent 测试并发场景下的线程安全
func TestStreamViewerTracker_Concurrent(t *testing.T) {
	tracker := NewStreamViewerTracker()
	var wg sync.WaitGroup

	// 并发添加和移除 viewer
	for i := 0; i < 100; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			tracker.AddViewer("stream-1")
		}()
		go func() {
			defer wg.Done()
			tracker.RemoveViewer("stream-1")
		}()
	}

	wg.Wait()

	// 最终结果应该是非负的
	count := tracker.GetViewerCount("stream-1")
	assert.GreaterOrEqual(t, count, 0, "viewer count should not be negative")
}
