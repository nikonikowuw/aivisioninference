// Package service 提供业务逻辑层
package service

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// ServiceStatus 服务状态
type ServiceStatus struct {
	Name    string `json:"name"`
	Status  string `json:"status"` // running/stopped/error/unknown
	PID     int    `json:"pid,omitempty"`
	Port    int    `json:"port,omitempty"`
	Message string `json:"message,omitempty"`
}

// ServiceStatusDetector 服务状态检测器
//
// 关键设计：使用业务已经在使用的客户端（*redis.Client、*gorm.DB）做存活性探测，
// 而不是另起一条 TCP 连接。这样探测结果与实际业务连接完全一致 ——
// 只要业务跑得起来，探测必然显示 running；只要任一连接出问题，探测立刻反映。
type ServiceStatusDetector struct {
	rdb                   *redis.Client
	db                    *gorm.DB
	zlmAPIURL             string
	cppSocketPath          string
	engineMetricsChecker  EngineMetricsChecker
}

// NewServiceStatusDetector 创建服务状态检测器
func NewServiceStatusDetector(
	rdb *redis.Client,
	db *gorm.DB,
	zlmAPIURL, cppSocketPath string,
) *ServiceStatusDetector {
	return &ServiceStatusDetector{
		rdb:           rdb,
		db:            db,
		zlmAPIURL:     zlmAPIURL,
		cppSocketPath: cppSocketPath,
	}
}

// DetectAll 检测所有服务状态
func (d *ServiceStatusDetector) DetectAll() []ServiceStatus {
	services := make([]ServiceStatus, 5)

	// Go 管理端 (自身)
	services[0] = d.detectGoServer()

	// C++ 推理引擎
	services[1] = d.detectCppEngine()

	// ZLMediaKit
	services[2] = d.detectZLM()

	// Redis - 使用业务已在用的 redis.Client.Ping
	services[3] = d.detectRedis()

	// PostgreSQL - 使用业务已在用的 gorm 客户端 Exec "SELECT 1"
	services[4] = d.detectPostgreSQL()

	return services
}

// detectGoServer 检测 Go 管理端状态
func (d *ServiceStatusDetector) detectGoServer() ServiceStatus {
	// Go 管理端是当前进程，所以总是 running
	return ServiceStatus{
		Name:   "go_server",
		Status: "running",
		PID:    os.Getpid(),
	}
}

// EngineMetricsChecker 引擎指标检查器接口（由 EngineMetricsStore 实现）
type EngineMetricsChecker interface {
	IsStale() bool
	GetLastUpdateTime() time.Time
	GetUpdateCount() uint64
}

// SetEngineMetricsChecker 设置引擎指标检查器（用于深度检测）
func (d *ServiceStatusDetector) SetEngineMetricsChecker(checker EngineMetricsChecker) {
	d.engineMetricsChecker = checker
}

// detectCppEngine 检测 C++ 推理引擎状态
// 增强检测：从仅 socket 连通性检测升级为基于 IPC 心跳 + EngineMetricsStore 指标新鲜度的深度检测
func (d *ServiceStatusDetector) detectCppEngine() ServiceStatus {
	if d.cppSocketPath == "" {
		return ServiceStatus{
			Name:    "cpp_engine",
			Status:  "unknown",
			Message: "socket path not configured",
		}
	}

	// 1. 检测 UDS socket 是否存在
	conn, err := net.DialTimeout("unix", d.cppSocketPath, 2*time.Second)
	if err != nil {
		return ServiceStatus{
			Name:    "cpp_engine",
			Status:  "stopped",
			Message: truncate(fmt.Sprintf("socket connect failed: %v", err), 200),
		}
	}
	_ = conn.Close()

	// 2. 如果有 EngineMetricsChecker，进行深度检测
	if d.engineMetricsChecker != nil {
		if d.engineMetricsChecker.IsStale() {
			lastUpdate := d.engineMetricsChecker.GetLastUpdateTime()
			return ServiceStatus{
				Name:   "cpp_engine",
				Status: "error",
				Message: truncate(fmt.Sprintf(
					"socket connected but no metrics update for >15s (last: %s, count: %d)",
					lastUpdate.Format(time.RFC3339),
					d.engineMetricsChecker.GetUpdateCount()), 200),
			}
		}

		return ServiceStatus{
			Name:   "cpp_engine",
			Status: "running",
			Message: fmt.Sprintf("metrics active, updates: %d",
				d.engineMetricsChecker.GetUpdateCount()),
		}
	}

	// 3. 降级：仅 socket 连通性检测
	return ServiceStatus{
		Name:   "cpp_engine",
		Status: "running",
		Message: "socket connected, metrics check not available",
	}
}

// detectZLM 检测 ZLMediaKit 状态
func (d *ServiceStatusDetector) detectZLM() ServiceStatus {
	if d.zlmAPIURL == "" {
		return ServiceStatus{
			Name:    "zlm",
			Status:  "unknown",
			Message: "API URL not configured",
		}
	}

	// 简单 TCP 探测 ZLM HTTP API 端口
	// 实际生产环境可以替换为调用 ZLM 的 /index/api/getSnap 或心跳接口
	host := stripScheme(d.zlmAPIURL)
	if host == "" {
		host = "127.0.0.1:80"
	}
	if !strings.Contains(host, ":") {
		host = host + ":80"
	}
	conn, err := net.DialTimeout("tcp", host, 2*time.Second)
	if err != nil {
		return ServiceStatus{
			Name:    "zlm",
			Status:  "stopped",
			Message: truncate(fmt.Sprintf("connect %s failed: %v", host, err), 200),
		}
	}
	_ = conn.Close()

	port := 80
	if _, p, ok := splitHostPort(host); ok {
		port = p
	}
	return ServiceStatus{
		Name:   "zlm",
		Status: "running",
		Port:   port,
	}
}

// detectRedis 使用业务已经在用的 *redis.Client.Ping 探测 Redis
// 这是关键修复：探测与实际业务连接复用同一条路径，避免了 "业务能用但探测显示 stopped" 的问题
func (d *ServiceStatusDetector) detectRedis() ServiceStatus {
	if d.rdb == nil {
		return ServiceStatus{
			Name:    "redis",
			Status:  "unknown",
			Message: "redis client not initialized",
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := d.rdb.Ping(ctx).Err(); err != nil {
		return ServiceStatus{
			Name:    "redis",
			Status:  "stopped",
			Message: truncate(fmt.Sprintf("ping failed: %v", err), 200),
		}
	}

	port := 6379
	if d.rdb.Options() != nil && d.rdb.Options().Addr != "" {
		if _, p, ok := splitHostPort(d.rdb.Options().Addr); ok {
			port = p
		}
	}
	return ServiceStatus{
		Name:   "redis",
		Status: "running",
		Port:   port,
	}
}

// detectPostgreSQL 使用业务已经在用的 *gorm.DB 探测 PostgreSQL
func (d *ServiceStatusDetector) detectPostgreSQL() ServiceStatus {
	if d.db == nil {
		return ServiceStatus{
			Name:    "postgresql",
			Status:  "unknown",
			Message: "gorm db not initialized",
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var result int
	if err := d.db.WithContext(ctx).Raw("SELECT 1").Scan(&result).Error; err != nil {
		return ServiceStatus{
			Name:    "postgresql",
			Status:  "stopped",
			Message: truncate(fmt.Sprintf("select 1 failed: %v", err), 200),
		}
	}

	port := 5432
	if addr := d.dsnHint(); addr != "" {
		if _, p, ok := splitHostPort(addr); ok {
			port = p
		}
	}
	return ServiceStatus{
		Name:   "postgresql",
		Status: "running",
		Port:   port,
	}
}

// dsnHint 尝试从 gorm 拿到 DSN 字符串（用于显示连接端口）
// 不同 Dialector 实现差异较大，这里只做尽力而为，不影响探测结果
func (d *ServiceStatusDetector) dsnHint() string {
	type dsnProvider interface {
		DSN() string
	}
	if p, ok := any(d.db.Dialector).(dsnProvider); ok {
		return p.DSN()
	}
	return ""
}

// splitHostPort 解析 host:port
func splitHostPort(addr string) (string, int, bool) {
	idx := strings.LastIndex(addr, ":")
	if idx < 0 {
		return addr, 0, false
	}
	host := addr[:idx]
	var port int
	if _, err := fmt.Sscanf(addr[idx+1:], "%d", &port); err != nil {
		return host, 0, false
	}
	return host, port, true
}

// stripScheme 去掉 http:// https:// 等 scheme 前缀
func stripScheme(s string) string {
	if i := strings.Index(s, "://"); i >= 0 {
		return s[i+3:]
	}
	return s
}

// truncate 截断长消息，避免 JSON 过大
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
