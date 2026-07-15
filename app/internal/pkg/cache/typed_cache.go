package cache

import (
	"context"
	"encoding/json"
	"time"
)

// TypedCache 是 Cache 接口的泛型包装，自动处理 JSON 序列化/反序列化，
// 让调用方以原生 Go 类型操作缓存数据。
type TypedCache[T any] struct {
	store Cache
	ttl   time.Duration
}

// NewTypedCache 创建泛型缓存实例。ttl=0 表示条目永不过期（需主动删除）。
func NewTypedCache[T any](store Cache, ttl time.Duration) *TypedCache[T] {
	return &TypedCache[T]{store: store, ttl: ttl}
}

// Get 反序列化并返回 key 对应的值。未命中时返回 ErrCacheMiss。
func (c *TypedCache[T]) Get(ctx context.Context, key string) (*T, error) {
	data, err := c.store.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	var val T
	if err := json.Unmarshal(data, &val); err != nil {
		return nil, err
	}
	return &val, nil
}

// Set 序列化并存储值。覆盖已有 key。
func (c *TypedCache[T]) Set(ctx context.Context, key string, val *T) error {
	data, err := json.Marshal(val)
	if err != nil {
		return err
	}
	return c.store.Set(ctx, key, data, c.ttl)
}

// Del 删除指定 key。条目不存在时返回 nil。
func (c *TypedCache[T]) Del(ctx context.Context, key string) error {
	return c.store.Del(ctx, key)
}
