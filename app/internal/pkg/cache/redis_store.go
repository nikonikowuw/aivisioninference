// Package cache provides Redis client connection management.
// It also provides an in-memory cache implementation for fallback scenarios.
package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisCache implements the Cache interface using a Redis backend.
type RedisCache struct {
	client *redis.Client
}

// NewRedisCache creates a new RedisCache backed by the given Redis client.
func NewRedisCache(client *redis.Client) *RedisCache {
	return &RedisCache{client: client}
}

// Get retrieves the value for the given key from Redis.
func (c *RedisCache) Get(ctx context.Context, key string) ([]byte, error) {
	data, err := c.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, ErrCacheMiss
	}
	return data, err
}

// Set stores a value with the given key and TTL in Redis.
func (c *RedisCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return c.client.Set(ctx, key, value, ttl).Err()
}

// Del removes the value for the given key from Redis.
func (c *RedisCache) Del(ctx context.Context, key string) error {
	return c.client.Del(ctx, key).Err()
}
