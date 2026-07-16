package model

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// EdgeNodeMetrics 边缘节点系统指标时序数据模型
// 每次心跳上报时记录一次快照，按时间戳存储
// 用于历史趋势分析、告警引擎评估和前端图表展示
type EdgeNodeMetrics struct {
	BaseModel
	NodeID    string    `gorm:"type:uuid;not null;index:idx_metrics_node_created,priority:1;comment:关联边缘节点ID" json:"node_id"`
	CreatedAt time.Time `gorm:"index:idx_metrics_node_created,priority:2;comment:记录时间" json:"created_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"` // soft delete

	// CPU
	CPUUsage    float64 `gorm:"not null;default:0;comment:CPU使用率 0-100" json:"cpu_usage"`
	CPULoad1m   float64 `gorm:"not null;default:0;comment:1分钟负载" json:"cpu_load_1m"`
	CPULoad5m   float64 `gorm:"not null;default:0;comment:5分钟负载" json:"cpu_load_5m"`
	CPULoad15m  float64 `gorm:"not null;default:0;comment:15分钟负载" json:"cpu_load_15m"`

	// Memory
	MemoryUsage float64 `gorm:"not null;default:0;comment:内存使用率 0-100" json:"memory_usage"`
	MemoryUsed  int64   `gorm:"not null;default:0;comment:已用内存(字节)" json:"memory_used"`
	MemoryTotal int64   `gorm:"not null;default:0;comment:总内存(字节)" json:"memory_total"`

	// Disk (JSON array: [{"path":"/","total":1e9,"used":5e8,"percent":50.0}])
	DiskUsage datatypes.JSON `gorm:"type:jsonb;not null;default:'[]';comment:磁盘使用情况" json:"disk_usage"`

	// Network
	NetRxBytes  int64   `gorm:"not null;default:0;comment:累计接收字节数" json:"net_rx_bytes"`
	NetTxBytes  int64   `gorm:"not null;default:0;comment:累计发送字节数" json:"net_tx_bytes"`
	NetRxSpeed  float64 `gorm:"not null;default:0;comment:接收速率(bytes/s)" json:"net_rx_speed"`
	NetTxSpeed  float64 `gorm:"not null;default:0;comment:发送速率(bytes/s)" json:"net_tx_speed"`

	// System
	Uptime       int64   `gorm:"not null;default:0;comment:运行时长(秒)" json:"uptime"`
	ProcessCount int     `gorm:"not null;default:0;comment:进程数" json:"process_count"`
	ThreadCount  int     `gorm:"not null;default:0;comment:线程数" json:"thread_count"`
	Temperature  float64 `gorm:"not null;default:0;comment:核心温度(摄氏度)" json:"temperature"`

	// Accelerator (NPU/GPU)
	AcceleratorUtilization  float64 `gorm:"not null;default:0;comment:加速器使用率 0-100" json:"accelerator_utilization"`
	AcceleratorMetricsValid bool    `gorm:"not null;default:false;comment:加速器指标是否有效" json:"accelerator_metrics_valid"`

	// Engine-specific
	WorkerCount       int `gorm:"not null;default:0;comment:工作线程数" json:"worker_count"`
	IdleWorkerCount   int `gorm:"not null;default:0;comment:空闲工作线程数" json:"idle_worker_count"`
	ActiveStreamCount int `gorm:"not null;default:0;comment:活跃流数" json:"active_stream_count"`
	DecodeSessions    int `gorm:"not null;default:0;comment:解码会话数" json:"decode_sessions"`
	EncodeSessions    int `gorm:"not null;default:0;comment:编码会话数" json:"encode_sessions"`
	CurrentLoad       int `gorm:"not null;default:0;comment:当前负载(任务数)" json:"current_load"`
	EngineVersion     string `gorm:"type:varchar(100);comment:引擎版本" json:"engine_version"`
	HALPlatform       string `gorm:"type:varchar(100);comment:HAL平台" json:"hal_platform"`
}

// TableName 指定表名
func (EdgeNodeMetrics) TableName() string {
	return "edge_node_metrics"
}

// SortableFields 返回允许排序的字段列表
func (EdgeNodeMetrics) SortableFields() []string {
	return []string{"created_at", "cpu_usage", "memory_usage", "uptime"}
}
