package model

import (
	"time"

	"gorm.io/datatypes"
)

// EdgeNodeMetrics 边缘节点时序指标（每次心跳记录一条）
type EdgeNodeMetrics struct {
	BaseModel
	NodeID    string    `gorm:"type:char(36);not null;comment:边缘节点ID" json:"node_id"`
	CreatedAt time.Time `gorm:"not null;comment:记录时间" json:"created_at"`

	CPUUsage   float64 `gorm:"type:double precision;not null;default:0;comment:CPU使用率(%)" json:"cpu_usage"`
	CPULoad1m  float64 `gorm:"type:double precision;not null;default:0;comment:CPU负载1分钟" json:"cpu_load_1m"`
	CPULoad5m  float64 `gorm:"type:double precision;not null;default:0;comment:CPU负载5分钟" json:"cpu_load_5m"`
	CPULoad15m float64 `gorm:"type:double precision;not null;default:0;comment:CPU负载15分钟" json:"cpu_load_15m"`

	MemoryUsage float64 `gorm:"type:double precision;not null;default:0;comment:内存使用率(%)" json:"memory_usage"`
	MemoryUsed  int64   `gorm:"type:bigint;not null;default:0;comment:已用内存(字节)" json:"memory_used"`
	MemoryTotal int64   `gorm:"type:bigint;not null;default:0;comment:总内存(字节)" json:"memory_total"`

	DiskUsage datatypes.JSON `gorm:"type:jsonb;not null;default:'[]';comment:磁盘挂载点使用情况" json:"disk_usage" swaggertype:"array,object"`

	NetRXBytes int64   `gorm:"type:bigint;not null;default:0;comment:网络累计接收(字节)" json:"net_rx_bytes"`
	NetTXBytes int64   `gorm:"type:bigint;not null;default:0;comment:网络累计发送(字节)" json:"net_tx_bytes"`
	NetRXSpeed float64 `gorm:"type:double precision;not null;default:0;comment:网络接收速率(字节/秒)" json:"net_rx_speed"`
	NetTXSpeed float64 `gorm:"type:double precision;not null;default:0;comment:网络发送速率(字节/秒)" json:"net_tx_speed"`

	Uptime        int64   `gorm:"type:bigint;not null;default:0;comment:运行时长(秒)" json:"uptime"`
	ProcessCount  int32   `gorm:"type:int;not null;default:0;comment:进程数" json:"process_count"`
	ThreadCount   int32   `gorm:"type:int;not null;default:0;comment:线程数" json:"thread_count"`
	Temperature   float64 `gorm:"type:double precision;not null;default:0;comment:核心温度(°C)" json:"temperature"`
	CurrentLoad   int32   `gorm:"type:int;not null;default:0;comment:当前负载(任务数)" json:"current_load"`
	EngineVersion string  `gorm:"type:varchar(100);comment:引擎版本" json:"engine_version"`
	HALPlatform   string  `gorm:"type:varchar(100);comment:HAL平台" json:"hal_platform"`
}

// TableName 指定表名
func (EdgeNodeMetrics) TableName() string {
	return "edge_node_metrics"
}

// SortableFields 返回允许排序的字段列表
func (EdgeNodeMetrics) SortableFields() []string {
	return []string{"created_at", "cpu_usage", "memory_usage", "uptime"}
}
