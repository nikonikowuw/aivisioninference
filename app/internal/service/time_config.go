// Package service 提供业务逻辑层
package service

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/beevik/ntp"
	"go.uber.org/zap"
	"golang.org/x/sys/unix"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/niko-admin/niko-admin/internal/model"
	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
)

// TimeConfigResponse 时间配置响应
type TimeConfigResponse struct {
	CurrentTime string     `json:"current_time"`
	Timezone    string     `json:"timezone"`
	TimeMode    string     `json:"time_mode"`
	NTPConfig   *NTPConfig `json:"ntp_config,omitempty"`
}

// NTPConfig NTP 配置
type NTPConfig struct {
	Enabled        bool            `json:"enabled"`
	Servers        []NTPServerInfo `json:"servers"`
	SyncInterval   int             `json:"sync_interval"`
	LastSyncAt     *time.Time      `json:"last_sync_at,omitempty"`
	LastSyncStatus string          `json:"last_sync_status,omitempty"`
}

// NTPServerInfo NTP 服务器信息
type NTPServerInfo struct {
	Host    string `json:"host"`
	Status  string `json:"status"` // reachable/unreachable
	Latency int64  `json:"latency_ms,omitempty"`
}

// ntpCacheTTL NTP 探测缓存有效期
const ntpCacheTTL = 30 * time.Second

// ntpCacheEntry NTP 探测缓存条目
type ntpCacheEntry struct {
	results  []NTPServerInfo
	expireAt time.Time
}

// TimeConfigService 时间配置服务
type TimeConfigService struct {
	db *gorm.DB

	// NTP 探测缓存（避免高频请求时频繁发起 NTP 探测）
	ntpMu    sync.RWMutex
	ntpCache map[string]*ntpCacheEntry // key = 逗号分隔的服务器列表
}

// NewTimeConfigService 创建时间配置服务
func NewTimeConfigService(db *gorm.DB) *TimeConfigService {
	return &TimeConfigService{
		db:       db,
		ntpCache: make(map[string]*ntpCacheEntry),
	}
}

// GetTimeConfig 获取时间配置
func (s *TimeConfigService) GetTimeConfig() (*TimeConfigResponse, error) {
	var config model.TimeConfig
	result := s.db.First(&config)
	if result.Error != nil && result.Error != gorm.ErrRecordNotFound {
		return nil, result.Error
	}

	if result.Error == gorm.ErrRecordNotFound {
		config = model.TimeConfig{
			ID:           "default",
			TimeMode:     model.TimeModeNTP,
			Timezone:     "Asia/Shanghai",
			NTPEnabled:   true,
			NTPServers:   datatypes.JSON(`["ntp.aliyun.com","cn.ntp.org.cn"]`),
			SyncInterval: 60,
		}
	}

	now := time.Now()
	servers := ntpServersFromConfig(config)
	serverInfos := s.testNTPServersReachabilityWithCache(servers)

	return &TimeConfigResponse{
		CurrentTime: now.Format(time.RFC3339),
		Timezone:    now.Location().String(),
		TimeMode:    config.TimeMode,
		NTPConfig: &NTPConfig{
			Enabled:        config.NTPEnabled,
			Servers:        serverInfos,
			SyncInterval:   config.SyncInterval,
			LastSyncAt:     config.LastSyncAt,
			LastSyncStatus: config.LastSyncStatus,
		},
	}, nil
}

// SetManualTime 手动设置时间
// 校验: 时间偏差不得超过 ±24 小时，防止误操作导致系统时间异常
func (s *TimeConfigService) SetManualTime(t time.Time) error {
	now := time.Now()
	diff := t.Sub(now)
	if diff > 24*time.Hour || diff < -24*time.Hour {
		return apperrors.New(apperrors.ErrTimeOutOfRange, "")
	}

	// 使用 unix 设置系统时间
	tv := unix.NsecToTimeval(t.UnixNano())

	if err := unix.Settimeofday(&tv); err != nil {
		return apperrors.New(apperrors.ErrSetTimeFailed, "")
	}

	return nil
}

// GetTimezone 获取时区
func (s *TimeConfigService) GetTimezone() (string, error) {
	// 从 /etc/timezone 读取
	data, err := os.ReadFile("/etc/timezone")
	if err == nil {
		return strings.TrimSpace(string(data)), nil
	}

	// 从 /etc/localtime 链接读取
	link, err := os.Readlink("/etc/localtime")
	if err == nil {
		// 格式: /usr/share/zoneinfo/Asia/Shanghai
		parts := strings.Split(link, "/zoneinfo/")
		if len(parts) > 1 {
			return parts[1], nil
		}
	}

	// 返回当前时区
	return time.Now().Location().String(), nil
}

// SetTimezone 设置时区
// 使用原子写入: 先写临时文件，再 rename 替换 /etc/localtime，防止中间状态丢失
func (s *TimeConfigService) SetTimezone(timezone string) error {
	// 验证时区有效性
	if _, err := time.LoadLocation(timezone); err != nil {
		zap.L().Warn("invalid timezone", zap.String("timezone", timezone), zap.Error(err))
		return apperrors.New(apperrors.ErrInvalidTimezone, "")
	}

	// 检查时区文件是否存在
	zoneinfoPath := fmt.Sprintf("/usr/share/zoneinfo/%s", timezone)
	if _, err := os.Stat(zoneinfoPath); os.IsNotExist(err) {
		return apperrors.New(apperrors.ErrTimezoneFileNotFound, "")
	}

	// 原子替换 /etc/localtime:
	// 1. 写入临时文件
	tmpPath := "/etc/localtime.tmp"
	if err := os.Remove(tmpPath); err != nil && !os.IsNotExist(err) {
		// 忽略不存在错误
	}

	// 2. 创建临时符号链接
	if err := os.Symlink(zoneinfoPath, tmpPath); err != nil {
		zap.L().Error("failed to create temp timezone symlink", zap.String("timezone", timezone), zap.Error(err))
		return apperrors.New(apperrors.ErrInvalidTimezone, "")
	}

	// 3. 原子 rename 替换
	if err := os.Rename(tmpPath, "/etc/localtime"); err != nil {
		// rename 失败时清理临时文件
		os.Remove(tmpPath)
		zap.L().Error("failed to replace localtime", zap.String("timezone", timezone), zap.Error(err))
		return apperrors.New(apperrors.ErrInvalidTimezone, "")
	}

	// 4. 更新 /etc/timezone (非关键，失败仅记录日志)
	if err := os.WriteFile("/etc/timezone", []byte(timezone), 0644); err != nil {
		zap.L().Warn("failed to update /etc/timezone", zap.Error(err))
	}

	// 5. 设置 TZ 环境变量（当前进程生效）
	os.Setenv("TZ", timezone)

	return nil
}

// GetNTPConfig 获取 NTP 配置
func (s *TimeConfigService) GetNTPConfig() (*NTPConfig, error) {
	var config model.TimeConfig
	result := s.db.First(&config)
	if result.Error != nil && result.Error != gorm.ErrRecordNotFound {
		return nil, result.Error
	}

	if result.Error == gorm.ErrRecordNotFound {
		return &NTPConfig{
			Enabled:      true,
			Servers:      []NTPServerInfo{},
			SyncInterval: 60,
		}, nil
	}

	servers := ntpServersFromConfig(config)
	serverInfos := s.testNTPServersReachabilityWithCache(servers)

	return &NTPConfig{
		Enabled:        config.NTPEnabled,
		Servers:        serverInfos,
		SyncInterval:   config.SyncInterval,
		LastSyncAt:     config.LastSyncAt,
		LastSyncStatus: config.LastSyncStatus,
	}, nil
}

// AddNTPServer 添加 NTP 服务器
func (s *TimeConfigService) AddNTPServer(host string) error {
	config, err := s.getOrCreateTimeConfig()
	if err != nil {
		return err
	}

	servers := ntpServersFromConfig(*config)
	for _, srv := range servers {
		if srv == host {
			return nil // 已存在
		}
	}

	// 添加新服务器
	servers = append(servers, host)
	serversJSON, _ := json.Marshal(servers)
	return s.db.Model(config).Update("ntp_servers", datatypes.JSON(serversJSON)).Error
}

// RemoveNTPServer 删除 NTP 服务器
func (s *TimeConfigService) RemoveNTPServer(host string) error {
	config, err := s.getOrCreateTimeConfig()
	if err != nil {
		return err
	}

	servers := ntpServersFromConfig(*config)
	newServers := make([]string, 0, len(servers))
	for _, srv := range servers {
		if srv != host {
			newServers = append(newServers, srv)
		}
	}

	serversJSON, _ := json.Marshal(newServers)
	return s.db.Model(config).Update("ntp_servers", datatypes.JSON(serversJSON)).Error
}

// TestNTPServers 测试 NTP 服务器连通性
func (s *TimeConfigService) TestNTPServers() ([]NTPServerInfo, error) {
	var config model.TimeConfig
	result := s.db.First(&config)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return []NTPServerInfo{}, nil
		}
		return nil, result.Error
	}

	servers := ntpServersFromConfig(config)
	return s.testNTPServersReachabilityWithCache(servers), nil
}

// SyncNTP 立即同步 NTP
func (s *TimeConfigService) SyncNTP() error {
	config, err := s.getOrCreateTimeConfig()
	if err != nil {
		return err
	}

	servers := ntpServersFromConfig(*config)
	var lastErr error
	for _, host := range servers {
		response, err := ntp.Query(host)
		if err != nil {
			lastErr = err
			continue
		}

		// 应用时间偏移
		now := time.Now().Add(response.ClockOffset)
		if err := s.SetManualTime(now); err != nil {
			lastErr = err
			continue
		}

		// 更新同步状态
		now = time.Now()
		s.db.Model(&config).Updates(map[string]interface{}{
			"last_sync_at":     now,
			"last_sync_status": "success",
		})

		return nil
	}

	// 所有服务器都失败
	now := time.Now()
	s.db.Model(&config).Updates(map[string]interface{}{
		"last_sync_at":     now,
		"last_sync_status": "failed",
	})

	zap.L().Warn("all NTP servers failed", zap.Error(lastErr))
	return apperrors.New(apperrors.ErrTimeSyncFailed, "")
}

// GetRecommendedNTPServers 获取推荐 NTP 服务器列表
func (s *TimeConfigService) GetRecommendedNTPServers() []string {
	return []string{
		"ntp.aliyun.com",
		"cn.ntp.org.cn",
		"ntp.tencent.com",
		"time.asia.apple.com",
		"pool.ntp.org",
		"time.google.com",
		"time.windows.com",
	}
}

// getOrCreateTimeConfig 获取或创建默认 NTP 配置
func (s *TimeConfigService) getOrCreateTimeConfig() (*model.TimeConfig, error) {
	var config model.TimeConfig
	result := s.db.First(&config)
	if result.Error == gorm.ErrRecordNotFound {
		// 数据库无记录时自动创建默认配置，与 GetTimeConfig 返回的默认值保持一致
		config = model.TimeConfig{
			ID:           "default",
			TimeMode:     model.TimeModeNTP,
			Timezone:     "Asia/Shanghai",
			NTPEnabled:   true,
			NTPServers:   datatypes.JSON(`["ntp.aliyun.com","cn.ntp.org.cn"]`),
			SyncInterval: 60,
		}
		if err := s.db.Create(&config).Error; err != nil {
			return nil, err
		}
		return &config, nil
	}
	if result.Error != nil {
		return nil, result.Error
	}
	return &config, nil
}

// ntpServersFromConfig 从 TimeConfig 中解析 NTP 服务器列表
func ntpServersFromConfig(config model.TimeConfig) []string {
	var servers []string
	if err := json.Unmarshal([]byte(config.NTPServers), &servers); err != nil {
		zap.L().Warn("failed to parse NTP servers config", zap.Error(err))
	}
	return servers
}

// parseNTPServers 将服务器地址列表转换为 NTPServerInfo 切片（状态默认 unknown）
func parseNTPServers(servers []string) []NTPServerInfo {
	result := make([]NTPServerInfo, 0, len(servers))
	for _, host := range servers {
		result = append(result, NTPServerInfo{
			Host:   host,
			Status: "unknown",
		})
	}
	return result
}

// testNTPServersReachabilityWithCache 带缓存的 NTP 探测
func (s *TimeConfigService) testNTPServersReachabilityWithCache(hosts []string) []NTPServerInfo {
	cacheKey := strings.Join(hosts, ",")

	// 尝试从缓存读取
	s.ntpMu.RLock()
	if entry, ok := s.ntpCache[cacheKey]; ok && time.Now().Before(entry.expireAt) {
		results := entry.results
		s.ntpMu.RUnlock()
		return results
	}
	s.ntpMu.RUnlock()

	// 缓存未命中，执行探测
	results := testNTPServersReachability(hosts)

	// 写入缓存
	s.ntpMu.Lock()
	s.ntpCache[cacheKey] = &ntpCacheEntry{
		results:  results,
		expireAt: time.Now().Add(ntpCacheTTL),
	}
	s.ntpMu.Unlock()

	return results
}

// testNTPServersReachability 并发测试 NTP 服务器连通性
func testNTPServersReachability(hosts []string) []NTPServerInfo {
	results := make([]NTPServerInfo, len(hosts))
	type result struct {
		index   int
		latency int64
		err     error
	}
	ch := make(chan result, len(hosts))

	for i, host := range hosts {
		go func(i int, host string) {
			start := time.Now()
			_, err := ntp.Query(host)
			ch <- result{index: i, latency: time.Since(start).Milliseconds(), err: err}
		}(i, host)
	}

	for range hosts {
		r := <-ch
		if r.err != nil {
			results[r.index] = NTPServerInfo{Host: hosts[r.index], Status: "unreachable"}
		} else {
			results[r.index] = NTPServerInfo{Host: hosts[r.index], Status: "reachable", Latency: r.latency}
		}
	}

	return results
}
