// HeartbeatStore 抽象了节点心跳存活性追踪，支持 Redis ZSET 和本地内存两种实现。
// 供 HandleHeartbeat 写入心跳时间戳、离线检测任务读取过期节点。
package service

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

const (
	redisHeartbeatZSet       = "aivision:edge:heartbeats"
	redisGB28181HeartbeatZSet = "aivision:gb28181:heartbeats"
)

// HeartbeatStore 节点心跳存活性存储接口。
type HeartbeatStore interface {
	// Record 记录节点的心跳时间戳。
	Record(ctx context.Context, nodeID string, ts time.Time) error
	// GetExpired 返回所有在 cutoff 之前心跳超时的节点 ID 列表。
	GetExpired(ctx context.Context, cutoff time.Time) ([]string, error)
	// Remove 移除节点的心跳记录。
	Remove(ctx context.Context, nodeID string) error
	// IsOnline 检查节点是否在线。
	IsOnline(ctx context.Context, nodeID string, timeout time.Duration) (bool, error)
	// BatchIsOnline 批量检查节点在线状态，返回 nodeID → online 映射。
	// 相比循环调用 IsOnline，可减少 Redis 往返次数。
	BatchIsOnline(ctx context.Context, nodeIDs []string, timeout time.Duration) (map[string]bool, error)
}

// NewHeartbeatStore 根据 Redis 客户端可用性创建边缘节点 HeartbeatStore。
func NewHeartbeatStore(rdb *redis.Client) HeartbeatStore {
	return NewHeartbeatStoreWithKey(rdb, redisHeartbeatZSet)
}

// NewGB28181HeartbeatStore 创建 GB28181 设备 HeartbeatStore。
func NewGB28181HeartbeatStore(rdb *redis.Client) HeartbeatStore {
	return NewHeartbeatStoreWithKey(rdb, redisGB28181HeartbeatZSet)
}

// NewHeartbeatStoreWithKey 创建指定 Redis key 的 HeartbeatStore。
func NewHeartbeatStoreWithKey(rdb *redis.Client, key string) HeartbeatStore {
	if rdb != nil {
		return &RedisHeartbeatStore{rdb: rdb, key: key}
	}
	zap.L().Warn("redis client not available, using in-memory heartbeat store (node liveness resets on restart)")
	return &MemoryHeartbeatStore{}
}

// ---------------------------------------------------------------------------
// RedisHeartbeatStore — 基于 Redis ZSET，适用于分布式多副本部署。
// ---------------------------------------------------------------------------

// RedisHeartbeatStore implements HeartbeatStore using a Redis sorted set.
type RedisHeartbeatStore struct {
	rdb *redis.Client
	key string // Redis key for the sorted set
}

func (s *RedisHeartbeatStore) Record(ctx context.Context, nodeID string, ts time.Time) error {
	return s.rdb.ZAdd(ctx, s.key, redis.Z{
		Score:  float64(ts.Unix()),
		Member: nodeID,
	}).Err()
}

func (s *RedisHeartbeatStore) GetExpired(ctx context.Context, cutoff time.Time) ([]string, error) {
	return s.rdb.ZRangeByScore(ctx, s.key, &redis.ZRangeBy{
		Min: "-inf",
		Max: strconv.FormatInt(cutoff.Unix(), 10),
	}).Result()
}

func (s *RedisHeartbeatStore) Remove(ctx context.Context, nodeID string) error {
	return s.rdb.ZRem(ctx, s.key, nodeID).Err()
}

func (s *RedisHeartbeatStore) IsOnline(ctx context.Context, nodeID string, timeout time.Duration) (bool, error) {
	score, err := s.rdb.ZScore(ctx, s.key, nodeID).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return false, nil
		}
		return false, err
	}
	lastHeartbeat := time.Unix(int64(score), 0)
	return time.Since(lastHeartbeat) <= timeout, nil
}

// BatchIsOnline 通过 ZMSCORE 一次查询所有节点的心跳分数，消除 N+1 Redis 往返。
func (s *RedisHeartbeatStore) BatchIsOnline(ctx context.Context, nodeIDs []string, timeout time.Duration) (map[string]bool, error) {
	if len(nodeIDs) == 0 {
		return map[string]bool{}, nil
	}
	scores, err := s.rdb.ZMScore(ctx, s.key, nodeIDs...).Result()
	if err != nil {
		return nil, err
	}
	result := make(map[string]bool, len(nodeIDs))
	for i, nodeID := range nodeIDs {
		score := scores[i]
		// ZMScore returns 0 for non-existent members; treat as offline
		if score == 0 {
			result[nodeID] = false
			continue
		}
		lastHeartbeat := time.Unix(int64(score), 0)
		result[nodeID] = time.Since(lastHeartbeat) <= timeout
	}
	return result, nil
}

// ---------------------------------------------------------------------------
// NewMemoryHeartbeatStore creates an in-memory HeartbeatStore for testing.
// It does not require a Redis connection and is intended for unit tests.
func NewMemoryHeartbeatStore() HeartbeatStore {
	return &MemoryHeartbeatStore{}
}

// MemoryHeartbeatStore — 基于 sync.Map，适用于开发/测试/无 Redis 环境。
// 注意：进程重启后存活性记录丢失，需要等待下一次心跳重建。
// ---------------------------------------------------------------------------

// MemoryHeartbeatStore implements HeartbeatStore using an in-memory map.
type MemoryHeartbeatStore struct {
	mu   sync.RWMutex
	data map[string]time.Time
}

func (s *MemoryHeartbeatStore) Record(_ context.Context, nodeID string, ts time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.data == nil {
		s.data = make(map[string]time.Time)
	}
	s.data[nodeID] = ts
	return nil
}

func (s *MemoryHeartbeatStore) GetExpired(_ context.Context, cutoff time.Time) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.data) == 0 {
		return nil, nil
	}

	var expired []string
	for id, last := range s.data {
		if last.Before(cutoff) {
			expired = append(expired, id)
		}
	}
	return expired, nil
}

func (s *MemoryHeartbeatStore) Remove(_ context.Context, nodeID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, nodeID)
	return nil
}

func (s *MemoryHeartbeatStore) IsOnline(_ context.Context, nodeID string, timeout time.Duration) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	last, ok := s.data[nodeID]
	if !ok {
		return false, nil
	}
	return time.Since(last) <= timeout, nil
}

func (s *MemoryHeartbeatStore) BatchIsOnline(_ context.Context, nodeIDs []string, timeout time.Duration) (map[string]bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make(map[string]bool, len(nodeIDs))
	for _, nodeID := range nodeIDs {
		last, ok := s.data[nodeID]
		if !ok {
			result[nodeID] = false
		} else {
			result[nodeID] = time.Since(last) <= timeout
		}
	}
	return result, nil
}
