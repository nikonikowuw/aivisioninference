package dto

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
	Metric      string `form:"metric" binding:"required,oneof=cpu_usage memory_usage cpu_load_1m cpu_load_5m cpu_load_15m memory_used memory_total disk_usage net_rx_bytes net_tx_bytes net_rx_speed net_tx_speed uptime process_count thread_count temperature current_load"` // 指标名
	From        string `form:"from" binding:"required"`                                                                                                                                                                                                                        // 起始时间 (RFC3339)
	To          string `form:"to" binding:"required"`                                                                                                                                                                                                                          // 结束时间 (RFC3339)
	Aggregation string `form:"aggregation" binding:"omitempty,oneof=avg max min"`                                                                                                                                                                                              // 聚合方式
	Interval    string `form:"interval" binding:"omitempty,oneof=1m 5m 15m 1h 6h 1d"`                                                                                                                                                                                          // 聚合窗口
	PageRequest
}
