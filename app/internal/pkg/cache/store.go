// Package cache 提供 Redis 客户端连接管理和本地内存缓存实现。
package cache

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/jellydator/ttlcache/v3"
)

// ErrCacheMiss 表示缓存未命中的哨兵错误。
var ErrCacheMiss = errors.New("cache: miss")

// DefaultMemoryCacheCapacity 是本地缓存的默认最大条目数。
const DefaultMemoryCacheCapacity uint64 = 50_000

// Cache 定义了统一的缓存读写接口，支持 Redis 和本地内存两种实现。
type Cache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Del(ctx context.Context, key string) error
}

// MemoryCache 是一个基于内存的 Cache 实现，支持 TTL 过期和定时清理。
type MemoryCache struct {
	store    *ttlcache.Cache[string, []byte]
	cancel   context.CancelFunc
	stopOnce sync.Once
	done     chan struct{}
}

type memoryCacheConfig struct {
	capacity uint64
}

// MemoryCacheOption 配置 MemoryCache 的容量等运行参数。
type MemoryCacheOption func(*memoryCacheConfig)

// WithMemoryCacheCapacity 设置最大条目数；传入 0 表示不限制容量。
func WithMemoryCacheCapacity(capacity uint64) MemoryCacheOption {
	return func(config *memoryCacheConfig) {
		config.capacity = capacity
	}
}

// NewMemoryCache creates a new memory cache.
// cleanupInterval specifies how often the janitor goroutine removes expired
// items from ttlcache's expiration queue. Pass 0 to disable periodic cleanup;
// expired items are still lazily evicted on Get.
//
// The caller MUST call Stop() when the cache is no longer needed to stop the
// background janitor goroutine. Failure to do so will leak a goroutine for the
// lifetime of the process.
// For context-aware lifecycle management, use NewMemoryCacheWithContext instead.
func NewMemoryCache(cleanupInterval time.Duration, options ...MemoryCacheOption) *MemoryCache {
	return newMemoryCache(context.Background(), cleanupInterval, options...)
}

// NewMemoryCacheWithContext is like NewMemoryCache but binds the janitor
// goroutine's lifecycle to ctx. When ctx is cancelled the janitor exits
// automatically, making it suitable for short-lived or context-scoped usage.
// The returned MemoryCache may still be used after ctx cancellation; only
// the background cleanup goroutine exits.
func NewMemoryCacheWithContext(ctx context.Context, cleanupInterval time.Duration, options ...MemoryCacheOption) *MemoryCache {
	return newMemoryCache(ctx, cleanupInterval, options...)
}

func newMemoryCache(ctx context.Context, cleanupInterval time.Duration, options ...MemoryCacheOption) *MemoryCache {
	config := memoryCacheConfig{capacity: DefaultMemoryCacheCapacity}
	for _, option := range options {
		option(&config)
	}

	storeOptions := []ttlcache.Option[string, []byte]{
		ttlcache.WithDisableTouchOnHit[string, []byte](),
	}
	if config.capacity > 0 {
		storeOptions = append(storeOptions, ttlcache.WithCapacity[string, []byte](config.capacity))
	}

	janitorCtx, cancel := context.WithCancel(ctx)
	c := &MemoryCache{
		store:  ttlcache.New(storeOptions...),
		cancel: cancel,
		done:   make(chan struct{}),
	}
	if cleanupInterval <= 0 {
		close(c.done)
		return c
	}

	go c.janitor(janitorCtx, cleanupInterval)
	return c
}

// Stop stops the background janitor goroutine. After Stop returns the janitor
// is guaranteed to have exited. It is safe to call multiple times.
// Callers MUST call Stop() (or use NewMemoryCacheWithContext) to avoid leaking
// the janitor goroutine.
func (c *MemoryCache) Stop() {
	c.stopOnce.Do(c.cancel)
	<-c.done
}

// Get 从内存缓存中获取指定键的值。
// ttlcache 内部已处理惰性淘汰，不需要额外调用 DeleteExpired。
func (c *MemoryCache) Get(_ context.Context, key string) ([]byte, error) {
	item := c.store.Get(key)
	if item == nil {
		return nil, ErrCacheMiss
	}
	value := item.Value()
	data := make([]byte, len(value))
	copy(data, value)
	return data, nil
}

// Set 将一个值存入内存缓存，并指定 TTL 过期时间（0 表示永不过期）。
func (c *MemoryCache) Set(_ context.Context, key string, value []byte, ttl time.Duration) error {
	copied := make([]byte, len(value))
	copy(copied, value)

	itemTTL := ttlcache.NoTTL
	if ttl > 0 {
		itemTTL = ttl
	}
	c.store.Set(key, copied, itemTTL)
	return nil
}

// Del 从内存缓存中删除指定键。
func (c *MemoryCache) Del(_ context.Context, key string) error {
	c.store.Delete(key)
	return nil
}

// janitor triggers ttlcache's expiration-queue cleanup on the configured interval.
func (c *MemoryCache) janitor(ctx context.Context, interval time.Duration) {
	defer close(c.done)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			c.store.DeleteExpired()
		case <-ctx.Done():
			return
		}
	}
}
