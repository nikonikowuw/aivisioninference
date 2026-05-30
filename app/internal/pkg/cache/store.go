// Package cache 提供 Redis 客户端连接管理和本地内存缓存实现。
package cache

import (
	"context"
	"errors"
	"sync"
	"time"
)

// ErrCacheMiss 表示缓存未命中的哨兵错误。
var ErrCacheMiss = errors.New("cache: miss")

// Cache 定义了统一的缓存读写接口，支持 Redis 和本地内存两种实现。
type Cache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Del(ctx context.Context, key string) error
}

type memoryItem struct {
	value     []byte
	expiresAt time.Time
}

// MemoryCache 是一个基于内存的 Cache 实现，支持 TTL 过期和定时清理。
type MemoryCache struct {
	mu     sync.RWMutex
	items  map[string]memoryItem
	stopCh chan struct{}
}

// NewMemoryCache creates a new memory cache.
// cleanupInterval specifies how often the janitor goroutine scans for expired
// items and removes them. Pass 0 to disable periodic cleanup (expired items
// are still lazily evicted on Get).
//
// The caller MUST call Stop() when the cache is no longer needed to stop the
// background janitor goroutine. Failure to do so will leak a goroutine for the
// lifetime of the process.
// For context-aware lifecycle management, use NewMemoryCacheWithContext instead.
func NewMemoryCache(cleanupInterval time.Duration) *MemoryCache {
	c := &MemoryCache{
		items:  make(map[string]memoryItem),
		stopCh: make(chan struct{}),
	}
	if cleanupInterval > 0 {
		go c.janitor(cleanupInterval)
	}
	return c
}

// NewMemoryCacheWithContext is like NewMemoryCache but binds the janitor
// goroutine's lifecycle to ctx. When ctx is cancelled the janitor exits
// automatically, making it suitable for short-lived or context-scoped usage.
// The returned MemoryCache may still be used after ctx cancellation; only
// the background cleanup goroutine exits.
func NewMemoryCacheWithContext(ctx context.Context, cleanupInterval time.Duration) *MemoryCache {
	c := &MemoryCache{
		items:  make(map[string]memoryItem),
		stopCh: make(chan struct{}),
	}
	if cleanupInterval > 0 {
		go c.janitorWithContext(ctx, cleanupInterval)
	}
	return c
}

// Stop stops the background janitor goroutine. After Stop returns the janitor
// is guaranteed to have exited. It is safe to call multiple times.
// Callers MUST call Stop() (or use NewMemoryCacheWithContext) to avoid leaking
// the janitor goroutine.
func (c *MemoryCache) Stop() {
	select {
	case <-c.stopCh:
	default:
		close(c.stopCh)
	}
}

// Get 从内存缓存中获取指定键的值（在读取时惰性删除已过期的条目）。
func (c *MemoryCache) Get(_ context.Context, key string) ([]byte, error) {
	c.mu.RLock()
	item, ok := c.items[key]
	c.mu.RUnlock()
	if !ok {
		return nil, ErrCacheMiss
	}
	if !item.expiresAt.IsZero() && time.Now().After(item.expiresAt) {
		c.mu.Lock()
		delete(c.items, key)
		c.mu.Unlock()
		return nil, ErrCacheMiss
	}
	data := make([]byte, len(item.value))
	copy(data, item.value)
	return data, nil
}

// Set 将一个值存入内存缓存，并指定 TTL 过期时间（0 表示永不过期）。
func (c *MemoryCache) Set(_ context.Context, key string, value []byte, ttl time.Duration) error {
	copied := make([]byte, len(value))
	copy(copied, value)

	item := memoryItem{value: copied}
	if ttl > 0 {
		item.expiresAt = time.Now().Add(ttl)
	}

	c.mu.Lock()
	c.items[key] = item
	c.mu.Unlock()
	return nil
}

// Del 从内存缓存中删除指定键。
func (c *MemoryCache) Del(_ context.Context, key string) error {
	c.mu.Lock()
	delete(c.items, key)
	c.mu.Unlock()
	return nil
}

// deleteExpired removes all expired items under a write lock.
func (c *MemoryCache) deleteExpired() {
	now := time.Now()
	c.mu.Lock()
	for k, item := range c.items {
		if !item.expiresAt.IsZero() && now.After(item.expiresAt) {
			delete(c.items, k)
		}
	}
	c.mu.Unlock()
}

// janitor runs periodic expired-item cleanup until Stop is called.
func (c *MemoryCache) janitor(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			c.deleteExpired()
		case <-c.stopCh:
			return
		}
	}
}

// janitorWithContext is like janitor but also exits when ctx is done.
// This allows context-based lifecycle management without an explicit Stop call.
func (c *MemoryCache) janitorWithContext(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			c.deleteExpired()
		case <-c.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}
