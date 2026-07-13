package dto

import (
	"time"
)

// NodeOverviewResponse 节点总览统计
type NodeOverviewResponse struct {
	Total      int64 `json:"total"`       // 总节点数
	Online     int64 `json:"online"`      // 在线节点数
	Offline    int64 `json:"offline"`     // 离线节点数
	ErrorCount int64 `json:"error_count"` // 错误节点数
	Disabled   int64 `json:"disabled"`    // 禁用节点数
}

// MetricsQueryRequest 指标查询请求
type MetricsQueryRequest struct {
	Metric      string `form:"metric" binding:"required,oneof=cpu_usage memory_usage cpu_load_1m cpu_load_5m cpu_load_15m memory_used memory_total disk_usage net_rx_bytes net_tx_bytes net_rx_speed net_tx_speed uptime process_count thread_count temperature current_load"`
	From        string `form:"from" binding:"required"`
	To          string `form:"to" binding:"required"`
	Aggregation string `form:"aggregation" binding:"omitempty,oneof=avg max min"`
	Interval    string `form:"interval" binding:"omitempty,oneof=1m 5m 15m 1h 6h 1d"`
	PageRequest
}

// MetricQueryRequest 指标查询请求参数（宽松校验，默认24h回溯）
type MetricQueryRequest struct {
	PageRequest
	Metric      string `form:"metric" binding:"required"`
	From        string `form:"from"` // RFC3339
	To          string `form:"to"`   // RFC3339
	Aggregation string `form:"aggregation" binding:"omitempty,oneof=avg max min"`
	Interval    string `form:"interval"` // e.g., "5m", "1h"
}

// GetFromTime 返回解析后的起始时间
func (r *MetricQueryRequest) GetFromTime() (time.Time, error) {
	if r.From == "" {
		return time.Now().Add(-24 * time.Hour), nil
	}
	return time.Parse(time.RFC3339, r.From)
}

// GetToTime 返回解析后的结束时间
func (r *MetricQueryRequest) GetToTime() (time.Time, error) {
	if r.To == "" {
		return time.Now(), nil
	}
	return time.Parse(time.RFC3339, r.To)
}

// MetricDataPoint 时间序列数据点
type MetricDataPoint struct {
	T string  `json:"t"` // ISO8601 timestamp
	V float64 `json:"v"` // value
}

// MetricQueryResponse 指标查询响应
type MetricQueryResponse struct {
	List  []MetricDataPoint `json:"list"`
	Total int64             `json:"total"`
}

// DiskUsageInfo 磁盘使用信息（匹配 C++ 侧 DiskUsageInfo）
type DiskUsageInfo struct {
	Path    string  `json:"path"`
	Total   int64   `json:"total"`
	Used    int64   `json:"used"`
	Percent float64 `json:"percent"`
}

// OverviewStats 边缘节点总览统计
type OverviewStats struct {
	Total      int `json:"total"`
	Online     int `json:"online"`
	Offline    int `json:"offline"`
	Error      int `json:"error"`
	AlertCount int `json:"alert_count"`
}
