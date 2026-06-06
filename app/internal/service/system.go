// Package service 提供业务逻辑层
package service

import (
	"context"
	"errors"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/niko-admin/niko-admin/internal/buildinfo"
	"github.com/niko-admin/niko-admin/internal/model"
	"github.com/niko-admin/niko-admin/internal/pkg/ipc"
	"github.com/niko-admin/niko-admin/internal/pkg/zlm"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// K-V 存储 key 常量
const (
	modelKeyDeviceModel = "device_model"
)

// SystemService 系统管理服务
type SystemService struct {
	db                  *gorm.DB
	rdb                 *redis.Client
	version             string
	metricsCollector    *MetricsCollector
	npuCollector        NPUCollector
	serviceDetector     *ServiceStatusDetector
	historyBuffer       *HistoryBuffer
	engineMetricsStore  *EngineMetricsStore
	zlmClient           ZLMAPI // ZLM API 抽象接口

	// 缓存实时指标
	mu            sync.RWMutex
	lastMetrics   *SystemMetrics
	lastResources *ResourceMetrics

	cancel context.CancelFunc // 用于停止采集协程
}

// NewSystemService 创建系统管理服务
func NewSystemService(
	db *gorm.DB,
	rdb *redis.Client,
	version string,
	zlmAPIURL, cppSocketPath string,
) *SystemService {
	if version == "" {
		version = buildinfo.Version
	}

	ctx, cancel := context.WithCancel(context.Background())

	s := &SystemService{
		db:               db,
		rdb:              rdb,
		version:          version,
		metricsCollector: NewMetricsCollector(),
		npuCollector:     NewNPUCollector(),
		serviceDetector:  NewServiceStatusDetector(rdb, db, zlmAPIURL, cppSocketPath),
		historyBuffer:    NewHistoryBuffer(60), // 保留 60 个数据点
		cancel:           cancel,
	}

	// 启动异步采集协程 (默认 2s 采集一次)
	s.startMetricsLoop(ctx, 2*time.Second)

	return s
}

// SetEngineMetricsStore 设置引擎指标存储（启动后注入）
func (s *SystemService) SetEngineMetricsStore(store *EngineMetricsStore) {
	s.engineMetricsStore = store
}

// GetEngineStatus 获取引擎全局指标
func (s *SystemService) GetEngineStatus() *EngineMetricsSummary {
	if s.engineMetricsStore == nil {
		return nil
	}
	return s.engineMetricsStore.GetSummary()
}

// GetStreamsStatus 获取各流推理指标
func (s *SystemService) GetStreamsStatus() []ipc.StreamMetricsSnapshot {
	if s.engineMetricsStore == nil {
		return nil
	}
	return s.engineMetricsStore.GetAllStreamMetrics()
}

// GetSIPStatus 获取 ZLM GB28181 SIP 服务运行状态
func (s *SystemService) GetSIPStatus() map[string]interface{} {
	status := map[string]interface{}{
		"enabled": false,
		"status":  "unknown",
	}
	// 从 ZLM 客户端查询 RTP 服务器列表
	if s.zlmClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if servers, err := s.zlmClient.ListRtpServer(ctx); err == nil {
			status["rtp_servers"] = len(servers)
			status["status"] = "running"
		} else {
			status["status"] = "error"
			status["error"] = err.Error()
		}
	}
	return status
}

// Close 停止采集协程，实现优雅关闭
func (s *SystemService) Close() error {
	if s.cancel != nil {
		s.cancel()
	}
	return nil
}

func (s *SystemService) startMetricsLoop(ctx context.Context, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		zap.L().Info("starting system metrics collection loop", zap.Duration("interval", interval))

		for {
			select {
			case <-ctx.Done():
				zap.L().Info("stopping system metrics collection loop")
				return
			case <-ticker.C:
				// 1. 采集实时指标 (CPU/内存)
				metrics, err := s.metricsCollector.CollectRealtimeMetrics()
				if err == nil {
					s.mu.Lock()
					s.lastMetrics = metrics
					s.mu.Unlock()

					// 记录到历史缓冲区
					s.historyBuffer.Add("cpu", metrics.CPU.UsagePercent, metrics.Timestamp)
					s.historyBuffer.Add("memory", metrics.Memory.UsagePercent, metrics.Timestamp)
				}

				// 2. 采集资源指标 (磁盘/NPU)
				resources, err := s.metricsCollector.CollectResourceMetrics(s.npuCollector)
				if err == nil {
					s.mu.Lock()
					s.lastResources = resources
					s.mu.Unlock()

					if resources.NPU.Supported {
						s.historyBuffer.Add("npu", resources.NPU.Usage, resources.Timestamp)
					}
				}
			}
		}
	}()
}

// GetRealtimeStatus 获取实时状态 (从缓存读取)
func (s *SystemService) GetRealtimeStatus() (*SystemMetrics, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.lastMetrics == nil {
		// 如果缓存还没准备好，同步采集一次
		return s.metricsCollector.CollectRealtimeMetrics()
	}

	return s.lastMetrics, nil
}

// GetResourceStatus 获取资源状态 (从缓存读取)
func (s *SystemService) GetResourceStatus() (*ResourceMetrics, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.lastResources == nil {
		return s.metricsCollector.CollectResourceMetrics(s.npuCollector)
	}

	return s.lastResources, nil
}

// GetServiceStatus 获取服务状态
func (s *SystemService) GetServiceStatus() (*ServiceStatusResponse, error) {
	services := s.serviceDetector.DetectAll()

	deviceModel := s.resolveDeviceModel()
	versions := s.collectVersions()

	// 获取时间状态 - 从 TimeConfig 表读 timezone（用户已在"时间配置"里设置的）
	timeStatus := s.collectTimeStatus()

	return &ServiceStatusResponse{
		DeviceModel: deviceModel,
		Versions:    versions,
		Time:        timeStatus,
		Services:    services,
		Timestamp:   time.Now(),
	}, nil
}

// GetStatusHistory 获取历史数据
func (s *SystemService) GetStatusHistory(metric string, duration time.Duration) ([]HistoryPoint, error) {
	return s.historyBuffer.Get(metric, duration), nil
}

// ZLMAPI ZLM 客户端最小接口（避免循环依赖）
type ZLMAPI interface {
	ListRtpServer(ctx context.Context) ([]zlm.RtpServerInfo, error)
}

// ServiceStatusResponse 服务状态响应
type ServiceStatusResponse struct {
	DeviceModel string           `json:"device_model"`
	Versions    VersionInfo      `json:"versions"`
	Time        TimeStatus       `json:"time"`
	Services    []ServiceStatus  `json:"services"`
	Timestamp   time.Time        `json:"timestamp"`
}

// VersionInfo 版本信息
type VersionInfo struct {
	App       string `json:"app"`
	Go        string `json:"go"`
	CPPEngine string `json:"cpp_engine"`
}

// TimeStatus 时间状态
type TimeStatus struct {
	Current  string `json:"current"`
	Timezone string `json:"timezone"`
	NTPStatus string `json:"ntp_status"`
}

// resolveDeviceModel 解析设备型号：
// 优先级：DB 中管理员设置的值 > 硬件自动检测 > 留空（前端用 "-" 显示）
//
// 设备型号有两种来源：
//  1. 管理员通过系统信息 API 手动设置（存于 ai_system_configs 表的 device_model key）
//  2. 通过 NPU/硬件自动检测（RK3576 / RK3588 / Huawei Ascend）
func (s *SystemService) resolveDeviceModel() string {
	// 1. 优先从 DB K-V 存储读
	if s.db != nil {
		var cfg model.AISystemConfig
		if err := s.db.Where("config_key = ?", modelKeyDeviceModel).First(&cfg).Error; err == nil {
			if v := strings.TrimSpace(cfg.ConfigValue); v != "" {
				return v
			}
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			zap.L().Warn("query device_model config failed", zap.Error(err))
		}
	}

	// 2. 硬件自动检测
	if hardware := detectHardware(); hardware != "" && hardware != "unknown" {
		return hardwareModelName(hardware)
	}

	// 3. 没检测到硬件且没设置过 —— 留空
	return ""
}

// hardwareModelName 硬件 ID -> 人类可读的型号名
func hardwareModelName(hardware string) string {
	switch hardware {
	case "rk3576":
		return "RK3576"
	case "rk3588":
		return "RK3588"
	case "ascend":
		return "Huawei Ascend"
	default:
		return hardware
	}
}

// collectVersions 收集版本信息
func (s *SystemService) collectVersions() VersionInfo {
	return VersionInfo{
		App:       s.version,              // 来自 main.go 的 Version（ldflags 注入）
		Go:        runtime.Version(),      // Go 运行时版本
		CPPEngine: s.detectCPPEngineVersion(),
	}
}

// detectCPPEngineVersion 探测 C++ 引擎版本（通过 UDS 握手或返回 "unknown"）
func (s *SystemService) detectCPPEngineVersion() string {
	// 暂保持简单：调用业务连接探测（如未来 C++ 引擎支持版本查询 API 时可补）
	return "unknown"
}

// collectTimeStatus 收集时间状态
// 优先从 TimeConfig 读用户配置的时区（"时间配置" Tab 里设置），否则用系统时区
func (s *SystemService) collectTimeStatus() TimeStatus {
	now := time.Now()
	tz := now.Location().String()

	if s.db != nil {
		var tc model.TimeConfig
		if err := s.db.First(&tc).Error; err == nil {
			if t := strings.TrimSpace(tc.Timezone); t != "" {
				// 尝试加载这个时区；如果加载失败则降级
				if loc, err := time.LoadLocation(t); err == nil {
					now = now.In(loc)
					tz = t
				} else {
					// 加载失败但 DB 里有值 —— 直接用字符串
					tz = t
				}
			}
		}
	}

	// ntp_status 暂用常量；如未来 NTP 服务能反馈状态可改成动态
	ntpStatus := "synced"

	return TimeStatus{
		Current:   now.Format(time.RFC3339),
		Timezone:  tz,
		NTPStatus: ntpStatus,
	}
}
