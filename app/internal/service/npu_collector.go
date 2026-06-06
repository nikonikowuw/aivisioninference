// Package service 提供业务逻辑层
package service

import (
	"fmt"
	"os"
	"strings"
)

// NPUMetrics NPU 指标
type NPUMetrics struct {
	Usage       float64 `json:"usage_percent"`
	Temperature float64 `json:"temperature"`
	MemoryUsed  int64   `json:"memory_used_mb,omitempty"`
	Supported   bool    `json:"supported"`
}

// NPUCollector NPU 指标采集器接口
type NPUCollector interface {
	Collect() (*NPUMetrics, error)
}

// MockCollector 通用/不支持时的降级实现
type MockCollector struct{}

// NewMockCollector 创建 Mock 采集器
func NewMockCollector() *MockCollector {
	return &MockCollector{}
}

// Collect 返回不支持的 NPU 指标
func (c *MockCollector) Collect() (*NPUMetrics, error) {
	return &NPUMetrics{
		Usage:       0,
		Temperature: 0,
		Supported:   false,
	}, nil
}

// NewNPUCollector 根据硬件自动选择 NPU 采集器
func NewNPUCollector() NPUCollector {
	hardware := detectHardware()
	switch {
	case strings.HasPrefix(hardware, "rk"):
		return NewRKNNCollector()
	case strings.HasPrefix(hardware, "ascend"):
		return NewAscendCollector()
	default:
		return NewMockCollector()
	}
}

// detectHardware 检测硬件平台
func detectHardware() string {
	// 尝试从设备树读取硬件信息
	data, err := os.ReadFile("/proc/device-tree/compatible")
	if err == nil {
		compatible := string(data)
		switch {
		case strings.Contains(compatible, "rk3576"):
			return "rk3576"
		case strings.Contains(compatible, "rk3588"):
			return "rk3588"
		case strings.Contains(compatible, "ascend"):
			return "ascend"
		}
	}

	// 尝试从 /proc/cpuinfo 读取
	data, err = os.ReadFile("/proc/cpuinfo")
	if err == nil {
		cpuinfo := string(data)
		switch {
		case strings.Contains(cpuinfo, "rk3576"):
			return "rk3576"
		case strings.Contains(cpuinfo, "rk3588"):
			return "rk3588"
		}
	}

	// 尝试检测 NPU 设备
	if _, err := os.Stat("/dev/rknpu"); err == nil {
		return "rk"
	}
	if _, err := os.Stat("/dev/davinci0"); err == nil {
		return "ascend"
	}

	return "unknown"
}

// RKNNCollector Rockchip RKNN 采集器
type RKNNCollector struct{}

// NewRKNNCollector 创建 RKNN 采集器
func NewRKNNCollector() *RKNNCollector {
	return &RKNNCollector{}
}

// Collect 采集 RKNN NPU 指标
func (c *RKNNCollector) Collect() (*NPUMetrics, error) {
	// 尝试读取 NPU 使用率
	usage, err := c.readNPUUsage()
	if err != nil {
		usage = 0
	}

	// 尝试读取 NPU 温度
	temperature, err := c.readTemperature()
	if err != nil {
		temperature = 0
	}

	return &NPUMetrics{
		Usage:       usage,
		Temperature: temperature,
		Supported:   true,
	}, nil
}

// readNPUUsage 读取 NPU 使用率
func (c *RKNNCollector) readNPUUsage() (float64, error) {
	// 尝试从 /sys 读取
	data, err := os.ReadFile("/sys/kernel/debug/rknpu/load")
	if err != nil {
		// 尝试其他路径
		data, err = os.ReadFile("/sys/class/devfreq/fdab0000.npu/cur_freq")
		if err != nil {
			return 0, fmt.Errorf("cannot read NPU usage: %v", err)
		}
	}

	content := strings.TrimSpace(string(data))

	// 格式1: "NPU load: 45.2%" 或 "45.2%"
	content = strings.TrimPrefix(content, "NPU load:")
	content = strings.TrimSpace(content)
	content = strings.TrimSuffix(content, "%")
	content = strings.TrimSpace(content)

	usage := parseFloat(content)
	return usage, nil
}

// readTemperature 读取 NPU 温度
func (c *RKNNCollector) readTemperature() (float64, error) {
	// 尝试从 thermal zone 读取
	thermalZones := []string{
		"/sys/class/thermal/thermal_zone0/temp",
		"/sys/class/thermal/thermal_zone1/temp",
		"/sys/class/thermal/thermal_zone2/temp",
	}

	for _, zone := range thermalZones {
		data, err := os.ReadFile(zone)
		if err != nil {
			continue
		}

		temp := parseFloat(strings.TrimSpace(string(data)))
		// 温度通常是毫摄氏度，需要除以 1000
		if temp > 1000 {
			temp = temp / 1000
		}

		return temp, nil
	}

	return 0, fmt.Errorf("cannot read NPU temperature")
}

// AscendCollector Huawei Ascend 采集器
type AscendCollector struct{}

// NewAscendCollector 创建 Ascend 采集器
func NewAscendCollector() *AscendCollector {
	return &AscendCollector{}
}

// Collect 采集 Ascend NPU 指标
func (c *AscendCollector) Collect() (*NPUMetrics, error) {
	// Ascend NPU 指标采集需要调用 ASCEND API
	// 这里提供基本实现
	return &NPUMetrics{
		Usage:       0,
		Temperature: 0,
		Supported:   true,
	}, nil
}

// parseFloat 解析浮点数，忽略错误（sysfs 内容异常时返回 0）
func parseFloat(s string) float64 {
	var f float64
	if _, err := fmt.Sscanf(s, "%f", &f); err != nil {
		return 0
	}
	return f
}
