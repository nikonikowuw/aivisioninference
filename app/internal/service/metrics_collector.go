// Package service 提供业务逻辑层
package service

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// SystemMetrics 系统指标
type SystemMetrics struct {
	CPU       CPUMetrics       `json:"cpu"`
	Memory    MemoryMetrics    `json:"memory"`
	Uptime    int64            `json:"uptime_seconds"`
	Tasks     TaskMetrics      `json:"tasks"`
	Timestamp time.Time        `json:"timestamp"`
}

// CPUMetrics CPU 指标
type CPUMetrics struct {
	UsagePercent float64 `json:"usage_percent"`
}

// cpuTick 表示一次 /proc/stat CPU 计数快照（用于两次采样计算差值）
type cpuTick struct {
	Idle  int64
	Total int64
}

// MemoryMetrics 内存指标
type MemoryMetrics struct {
	TotalMB     int64   `json:"total_mb"`
	UsedMB      int64   `json:"used_mb"`
	AvailableMB int64   `json:"available_mb"`
	UsagePercent float64 `json:"usage_percent"`
}

// ResourceMetrics 资源指标
type ResourceMetrics struct {
	NPU       NPUMetrics    `json:"npu"`
	Disks     []DiskMetrics `json:"disks"`
	Timestamp time.Time     `json:"timestamp"`
}

// DiskMetrics 磁盘指标
type DiskMetrics struct {
	MountPoint   string  `json:"mount_point"`
	TotalGB      float64 `json:"total_gb"`
	UsedGB       float64 `json:"used_gb"`
	AvailableGB  float64 `json:"available_gb"`
	UsagePercent float64 `json:"usage_percent"`
}

// TaskMetrics 任务指标
type TaskMetrics struct {
	Running int `json:"running"`
	Failed  int `json:"failed"`
}

// MetricsCollector 指标采集服务
type MetricsCollector struct {
	startTime time.Time
	lastCPUTick *cpuTick // 上一次 CPU 采样（Linux 两次采样差值计算使用率）
}

// NewMetricsCollector 创建指标采集服务
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		startTime: time.Now(),
	}
}

// CollectRealtimeMetrics 采集实时指标 (CPU/内存/运行时长/任务)
func (c *MetricsCollector) CollectRealtimeMetrics() (*SystemMetrics, error) {
	metrics := &SystemMetrics{
		Timestamp: time.Now(),
	}

	// CPU 使用率
	cpuUsage, err := c.getCPUUsage()
	if err == nil {
		metrics.CPU.UsagePercent = cpuUsage
	}

	// 内存使用率
	memMetrics, err := c.getMemoryMetrics()
	if err == nil {
		metrics.Memory = *memMetrics
	}

	// 运行时长
	metrics.Uptime = int64(time.Since(c.startTime).Seconds())

	return metrics, nil
}

// CollectResourceMetrics 采集资源指标 (NPU/磁盘/网络)
func (c *MetricsCollector) CollectResourceMetrics(npuCollector NPUCollector) (*ResourceMetrics, error) {
	metrics := &ResourceMetrics{
		Timestamp: time.Now(),
	}

	// NPU 指标
	if npuCollector != nil {
		npuMetrics, err := npuCollector.Collect()
		if err == nil && npuMetrics != nil {
			metrics.NPU = *npuMetrics
		}
	}

	// 磁盘指标
	disks, err := c.getDiskMetrics()
	if err == nil {
		metrics.Disks = disks
	}

	return metrics, nil
}

// getCPUUsage 获取 CPU 使用率
func (c *MetricsCollector) getCPUUsage() (float64, error) {
	switch runtime.GOOS {
	case "linux":
		return c.getCPUUsageLinux()
	case "darwin":
		return c.getCPUUsageDarwin()
	default:
		// 降级：使用 runtime 信息
		return 0, nil
	}
}

// getCPUUsageDarwin macOS 平台获取 CPU 使用率
func (c *MetricsCollector) getCPUUsageDarwin() (float64, error) {
	// 使用 sysctl 获取 CPU 使用率
	cmd := exec.Command("sysctl", "-n", "kern.cp_time")
	output, err := cmd.Output()
	if err != nil {
		// 降级：使用 top 命令
		return c.getCPUUsageDarwinFallback()
	}

	// 格式: user nice system idle intr
	fields := strings.Fields(string(output))
	if len(fields) < 5 {
		return c.getCPUUsageDarwinFallback()
	}

	var values [5]int64
	for i := 0; i < 5 && i < len(fields); i++ {
		values[i], _ = strconv.ParseInt(fields[i], 10, 64)
	}

	user := values[0]
	nice := values[1]
	system := values[2]
	idle := values[3]

	total := user + nice + system + idle
	if total == 0 {
		return 0, nil
	}

	used := user + nice + system
	usage := float64(used) / float64(total) * 100.0
	return usage, nil
}

// getCPUUsageDarwinFallback macOS 降级方案获取 CPU 使用率
func (c *MetricsCollector) getCPUUsageDarwinFallback() (float64, error) {
	cmd := exec.Command("top", "-l", "1", "-n", "0", "-s", "0")
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "CPU usage") || strings.Contains(line, "CPU Usage") {
			// 格式: CPU usage: 12.34% user, 5.67% sys, 82.0% idle
			parts := strings.Split(line, ",")
			for _, part := range parts {
				part = strings.TrimSpace(part)
				if strings.HasSuffix(part, "% idle") {
					idleStr := strings.TrimSuffix(part, "% idle")
					idleStr = strings.TrimSpace(idleStr)
					if idle, err := strconv.ParseFloat(idleStr, 64); err == nil {
						return 100.0 - idle, nil
					}
				}
			}
		}
	}

	return 0, fmt.Errorf("failed to parse CPU usage from top output")
}

// getCPUUsageLinux Linux 平台获取 CPU 使用率（两次采样差值法）
// 首次调用返回 0（尚无上一次采样），后续返回真实实时使用率
func (c *MetricsCollector) getCPUUsageLinux() (float64, error) {
	tick, err := readCPUTick()
	if err != nil {
		return 0, err
	}

	if c.lastCPUTick == nil {
		// 首次采样，记录快照，返回 0
		c.lastCPUTick = tick
		return 0, nil
	}

	// 计算差值
	deltaTotal := tick.Total - c.lastCPUTick.Total
	deltaIdle := tick.Idle - c.lastCPUTick.Idle

	// 更新采样快照
	c.lastCPUTick = tick

	if deltaTotal == 0 {
		return 0, nil
	}

	usage := float64(deltaTotal-deltaIdle) / float64(deltaTotal) * 100
	return usage, nil
}

// readCPUTick 读取 /proc/stat 第一行 CPU 计数
func readCPUTick() (*cpuTick, error) {
	file, err := os.Open("/proc/stat")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		return nil, fmt.Errorf("failed to read /proc/stat")
	}

	fields := strings.Fields(scanner.Text())
	if len(fields) < 8 || fields[0] != "cpu" {
		return nil, fmt.Errorf("invalid /proc/stat format")
	}

	// user, nice, system, idle, iowait, irq, softirq
	values := make([]int64, 7)
	for i := 0; i < 7; i++ {
		values[i], _ = strconv.ParseInt(fields[i+1], 10, 64)
	}

	idle := values[3] + values[4] // idle + iowait
	total := int64(0)
	for _, v := range values {
		total += v
	}

	return &cpuTick{Idle: idle, Total: total}, nil
}

// getMemoryMetrics 获取内存指标
func (c *MetricsCollector) getMemoryMetrics() (*MemoryMetrics, error) {
	switch runtime.GOOS {
	case "linux":
		return c.getMemoryMetricsLinux()
	case "darwin":
		return c.getMemoryMetricsDarwin()
	default:
		return c.getMemoryMetricsFallback()
	}
}

// getMemoryMetricsDarwin macOS 平台获取内存指标
func (c *MetricsCollector) getMemoryMetricsDarwin() (*MemoryMetrics, error) {
	// 使用 sysctl 获取物理内存
	cmd := exec.Command("sysctl", "-n", "hw.memsize")
	output, err := cmd.Output()
	if err != nil {
		return c.getMemoryMetricsFallback()
	}

	totalBytes, err := strconv.ParseInt(strings.TrimSpace(string(output)), 10, 64)
	if err != nil {
		return c.getMemoryMetricsFallback()
	}
	totalMB := totalBytes / 1024 / 1024

	// 使用 vm_stat 获取内存使用情况
	cmd = exec.Command("vm_stat")
	output, err = cmd.Output()
	if err != nil {
		return c.getMemoryMetricsFallback()
	}

	var pageSize int64 = 16384 // 默认 16KB
	var freePages, activePages, inactivePages, wiredPages int64

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "page size of") {
			// 解析页面大小
			parts := strings.Fields(line)
			for i, part := range parts {
				if part == "of" && i+1 < len(parts) {
					pageSize, _ = strconv.ParseInt(parts[i+1], 10, 64)
				}
			}
		}
		if strings.HasPrefix(line, "Pages free:") {
			fields := strings.Fields(line)
			if len(fields) >= 3 {
				val := strings.TrimSuffix(fields[2], ".")
				freePages, _ = strconv.ParseInt(val, 10, 64)
			}
		}
		if strings.HasPrefix(line, "Pages active:") {
			fields := strings.Fields(line)
			if len(fields) >= 3 {
				val := strings.TrimSuffix(fields[2], ".")
				activePages, _ = strconv.ParseInt(val, 10, 64)
			}
		}
		if strings.HasPrefix(line, "Pages inactive:") {
			fields := strings.Fields(line)
			if len(fields) >= 3 {
				val := strings.TrimSuffix(fields[2], ".")
				inactivePages, _ = strconv.ParseInt(val, 10, 64)
			}
		}
		if strings.HasPrefix(line, "Pages wired down:") {
			fields := strings.Fields(line)
			if len(fields) >= 4 {
				val := strings.TrimSuffix(fields[3], ".")
				wiredPages, _ = strconv.ParseInt(val, 10, 64)
			}
		}
	}

	usedMB := (activePages + wiredPages) * pageSize / 1024 / 1024
	availableMB := (freePages + inactivePages) * pageSize / 1024 / 1024

	var usagePercent float64
	if totalMB > 0 {
		usagePercent = float64(usedMB) / float64(totalMB) * 100
	}

	return &MemoryMetrics{
		TotalMB:      totalMB,
		UsedMB:       usedMB,
		AvailableMB:  availableMB,
		UsagePercent: usagePercent,
	}, nil
}

// getMemoryMetricsLinux Linux 平台获取内存指标 (优化为直接读取文件)
func (c *MetricsCollector) getMemoryMetricsLinux() (*MemoryMetrics, error) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var totalKB, availableKB int64
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		val, _ := strconv.ParseInt(fields[1], 10, 64)
		if strings.HasPrefix(line, "MemTotal:") {
			totalKB = val
		} else if strings.HasPrefix(line, "MemAvailable:") {
			availableKB = val
		}
		
		if totalKB > 0 && availableKB > 0 {
			break
		}
	}

	totalMB := totalKB / 1024
	availableMB := availableKB / 1024
	usedMB := totalMB - availableMB

	var usagePercent float64
	if totalMB > 0 {
		usagePercent = float64(usedMB) / float64(totalMB) * 100
	}

	return &MemoryMetrics{
		TotalMB:      totalMB,
		UsedMB:       usedMB,
		AvailableMB:  availableMB,
		UsagePercent: usagePercent,
	}, nil
}

// getMemoryMetricsFallback 降级获取内存指标
func (c *MetricsCollector) getMemoryMetricsFallback() (*MemoryMetrics, error) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	totalMB := int64(m.Sys / 1024 / 1024)
	usedMB := int64(m.Alloc / 1024 / 1024)
	availableMB := totalMB - usedMB

	var usagePercent float64
	if totalMB > 0 {
		usagePercent = float64(usedMB) / float64(totalMB) * 100
	}

	return &MemoryMetrics{
		TotalMB:      totalMB,
		UsedMB:       usedMB,
		AvailableMB:  availableMB,
		UsagePercent: usagePercent,
	}, nil
}

// getDiskMetrics 获取磁盘指标
func (c *MetricsCollector) getDiskMetrics() ([]DiskMetrics, error) {
	switch runtime.GOOS {
	case "linux":
		return c.getDiskMetricsLinux()
	case "darwin":
		return c.getDiskMetricsDarwin()
	default:
		return []DiskMetrics{}, nil
	}
}

// getDiskMetricsDarwin macOS 平台获取磁盘指标
func (c *MetricsCollector) getDiskMetricsDarwin() ([]DiskMetrics, error) {
	return c.getDiskMetricsStatfs()
}

// getDiskMetricsLinux Linux 平台获取磁盘指标
func (c *MetricsCollector) getDiskMetricsLinux() ([]DiskMetrics, error) {
	return c.getDiskMetricsStatfs()
}

// getDiskMetricsStatfs 使用 syscall.Statfs 跨平台获取磁盘指标
// 避免 df -kP 在不同 OS / 外部卷上的单位不一致问题（macOS 对 /Volumes/* 的
// 块大小与 / 不一致，会导致解析后的 GB 数值严重偏小）
func (c *MetricsCollector) getDiskMetricsStatfs() ([]DiskMetrics, error) {
	// 取所有挂载点（用 df -lP 仅做挂载点枚举，不再做单位解析）
	cmd := exec.Command("df", "-lP")
	output, err := cmd.Output()
	if err != nil {
		// df 不可用时降级：仅返回根目录
		return c.statfsDiskForPath("/"), nil
	}

	var disks []DiskMetrics
	seen := make(map[string]bool)
	lines := strings.Split(string(output), "\n")
	for i, line := range lines {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}
		mountPoint := fields[5]
		if !shouldReportMountPoint(mountPoint) {
			continue
		}
		if seen[mountPoint] {
			continue
		}
		if d, ok := c.statfsDisk(mountPoint); ok {
			disks = append(disks, d)
			seen[mountPoint] = true
		}
	}

	// 兜底：若一个挂载点都没拿到，保证至少有 /
	if len(disks) == 0 {
		disks = append(disks, c.statfsDiskForPath("/")...)
	}
	return disks, nil
}

// shouldReportMountPoint 过滤出主要挂载点
func shouldReportMountPoint(p string) bool {
	if p == "/" || strings.HasPrefix(p, "/data") || strings.HasPrefix(p, "/Volumes") {
		return true
	}
	return false
}

// statfsDisk 通过 syscall.Statfs 读取指定挂载点
func (c *MetricsCollector) statfsDisk(mountPoint string) (DiskMetrics, bool) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(mountPoint, &stat); err != nil {
		return DiskMetrics{}, false
	}
	return buildDiskMetrics(mountPoint, &stat), true
}

// statfsDiskForPath 遍历 stat 多个路径
func (c *MetricsCollector) statfsDiskForPath(path string) []DiskMetrics {
	if d, ok := c.statfsDisk(path); ok {
		return []DiskMetrics{d}
	}
	return nil
}

// buildDiskMetrics 从 Statfs_t 计算 GB 数值
func buildDiskMetrics(mountPoint string, stat *syscall.Statfs_t) DiskMetrics {
	// Bsize 是最优传输块大小，Blocks/Bfree/Bavail 单位都是 Bsize
	totalBytes := uint64(stat.Bsize) * uint64(stat.Blocks)
	freeBytes := uint64(stat.Bsize) * uint64(stat.Bavail) // 对普通用户可用的块
	usedBytes := totalBytes - freeBytes
	if usedBytes > totalBytes {
		usedBytes = totalBytes
	}

	const bytesPerGB = 1024.0 * 1024.0 * 1024.0
	totalGB := float64(totalBytes) / bytesPerGB
	usedGB := float64(usedBytes) / bytesPerGB
	availableGB := float64(freeBytes) / bytesPerGB

	var usagePercent float64
	if totalBytes > 0 {
		usagePercent = float64(usedBytes) / float64(totalBytes) * 100
	}

	return DiskMetrics{
		MountPoint:   mountPoint,
		TotalGB:      totalGB,
		UsedGB:       usedGB,
		AvailableGB:  availableGB,
		UsagePercent: usagePercent,
	}
}


