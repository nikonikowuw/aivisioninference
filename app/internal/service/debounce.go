package service

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// DebounceConfig 防抖配置
type DebounceConfig struct {
	// 防抖窗口 (默认 5s)
	Window time.Duration
	// Redis key 前缀
	KeyPrefix string
}

// DefaultDebounceConfig 返回默认配置
func DefaultDebounceConfig() DebounceConfig {
	return DebounceConfig{
		Window:    5 * time.Second,
		KeyPrefix: "debounce:alarm",
	}
}

// DebounceService 基于 Redis SETNX 的全局告警防抖服务
// 核心逻辑：基于设备 ID + 算法 ID + 告警事件类型组合生成原子 Key，
// 使用 SETNX + TTL 实现 5s 窗口内唯一推送。
type DebounceService struct {
	rdb    *redis.Client
	config DebounceConfig
	logger *zap.Logger
}

// NewDebounceService 创建防抖服务
func NewDebounceService(rdb *redis.Client, config DebounceConfig) *DebounceService {
	return &DebounceService{
		rdb:    rdb,
		config: config,
		logger: zap.L().With(zap.String("component", "debounce")),
	}
}

// DebounceKey 生成防抖 Key
// 组合: {prefix}:{device_id}:{algo_name}:{alarm_type}
func (s *DebounceService) DebounceKey(deviceID, algoName, alarmType string) string {
	return fmt.Sprintf("%s:%s:%s:%s", s.config.KeyPrefix, deviceID, algoName, alarmType)
}

// TryAcquire 尝试获取防抖许可
// 返回 true 表示该事件在窗口期内首次出现，可以继续处理。
// 返回 false 表示该事件在窗口期内已处理过，应丢弃推送。
func (s *DebounceService) TryAcquire(ctx context.Context, deviceID, algoName, alarmType string) (bool, error) {
	key := s.DebounceKey(deviceID, algoName, alarmType)

	// SETNX: 仅当 key 不存在时设置成功
	ok, err := s.rdb.SetNX(ctx, key, "1", s.config.Window).Result()
	if err != nil {
		s.logger.Warn("debounce SETNX failed",
			zap.String("key", key),
			zap.Error(err))
		// 防抖失败时，保守起见允许通过（不丢失告警）
		return true, nil
	}

	if ok {
		s.logger.Debug("debounce acquired",
			zap.String("key", key),
			zap.Duration("window", s.config.Window))
	}

	return ok, nil
}

// TryAcquireAlarm 针对告警事件的便捷方法
func (s *DebounceService) TryAcquireAlarm(ctx context.Context, deviceID, algoName, alarmType string) bool {
	ok, _ := s.TryAcquire(ctx, deviceID, algoName, alarmType)
	return ok
}

// Release 手动释放防抖锁（提前释放窗口）
func (s *DebounceService) Release(ctx context.Context, deviceID, algoName, alarmType string) error {
	key := s.DebounceKey(deviceID, algoName, alarmType)
	return s.rdb.Del(ctx, key).Err()
}
