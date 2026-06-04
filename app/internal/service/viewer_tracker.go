package service

import "sync"

// StreamViewerTracker tracks viewer counts per stream for auto-disconnect.
// 线程安全：使用 RWMutex 保护并发访问。
type StreamViewerTracker struct {
	mu      sync.RWMutex
	viewers map[string]int // streamKey -> viewer count
}

// NewStreamViewerTracker creates a new StreamViewerTracker.
func NewStreamViewerTracker() *StreamViewerTracker {
	return &StreamViewerTracker{
		viewers: make(map[string]int),
	}
}

// AddViewer increments viewer count for a stream.
// Returns the new count.
func (t *StreamViewerTracker) AddViewer(streamKey string) int {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.viewers[streamKey]++
	return t.viewers[streamKey]
}

// RemoveViewer decrements viewer count for a stream.
// Returns the new count (0 if no one watching).
func (t *StreamViewerTracker) RemoveViewer(streamKey string) int {
	t.mu.Lock()
	defer t.mu.Unlock()
	if count, ok := t.viewers[streamKey]; ok && count > 0 {
		t.viewers[streamKey]--
		if t.viewers[streamKey] <= 0 {
			delete(t.viewers, streamKey)
			return 0
		}
		return t.viewers[streamKey]
	}
	return 0
}

// GetViewerCount returns the current viewer count for a stream.
func (t *StreamViewerTracker) GetViewerCount(streamKey string) int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.viewers[streamKey]
}

// HasViewers checks if any viewers are watching a stream.
func (t *StreamViewerTracker) HasViewers(streamKey string) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.viewers[streamKey] > 0
}
