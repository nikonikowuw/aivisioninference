package model

import (
	"time"

	"gorm.io/gorm"
)

// EdgeNodeEngineMetrics represents the timeseries of C++ Inference Engine business/telemetry metrics.
type EdgeNodeEngineMetrics struct {
	BaseModel
	NodeID    string         `gorm:"type:uuid;not null;index:idx_engine_metrics_node_created,priority:1;comment:关联边缘节点ID" json:"node_id"`
	CreatedAt time.Time      `gorm:"index:idx_engine_metrics_node_created,priority:2;comment:记录时间" json:"created_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"` // soft delete or physical delete fallback

	ActiveStreamCount       uint32  `gorm:"not null;default:0;comment:活跃流数" json:"active_stream_count"`
	DMAUsedBytes            uint64  `gorm:"not null;default:0;comment:DMA已用字节" json:"dma_used_bytes"`
	DMATotalBytes           uint64  `gorm:"not null;default:0;comment:DMA总容量" json:"dma_total_bytes"`
	NPUUsedBytes            uint64  `gorm:"not null;default:0;comment:NPU已用字节" json:"npu_used_bytes"`
	NPUTotalBytes           uint64  `gorm:"not null;default:0;comment:NPU总容量" json:"npu_total_bytes"`
	WorkerCount             uint32  `gorm:"not null;default:0;comment:工作线程数" json:"worker_count"`
	IdleWorkerCount         uint32  `gorm:"not null;default:0;comment:空闲工作线程数" json:"idle_worker_count"`
	DecodeSessions          uint32  `gorm:"not null;default:0;comment:解码会话数" json:"decode_sessions"`
	EncodeSessions          uint32  `gorm:"not null;default:0;comment:编码会话数" json:"encode_sessions"`
	DecodeSlotsUsed         uint32  `gorm:"not null;default:0;comment:已占用解码槽" json:"decode_slots_used"`
	EncodeSlotsUsed         uint32  `gorm:"not null;default:0;comment:已占用编码槽" json:"encode_slots_used"`
	EgressBPS               uint64  `gorm:"not null;default:0;comment:媒体出口带宽(bps)" json:"egress_bps"`
	PreviewPipelineCount    uint32  `gorm:"not null;default:0;comment:仅预览Pipeline数" json:"preview_pipeline_count"`
	InferencePipelineCount  uint32  `gorm:"not null;default:0;comment:仅推理Pipeline数" json:"inference_pipeline_count"`
	MixedPipelineCount      uint32  `gorm:"not null;default:0;comment:同时预览和推理的Pipeline数" json:"mixed_pipeline_count"`
	MediaMetricsValid       bool    `gorm:"not null;default:false;comment:媒体指标是否有效" json:"media_metrics_valid"`
	PreviewCapacity         uint32  `gorm:"not null;default:0;comment:最大并发预览数" json:"preview_capacity"`
	PreviewInUse            uint32  `gorm:"not null;default:0;comment:当前预览使用数" json:"preview_in_use"`
	PreviewCapacityValid    bool    `gorm:"not null;default:false;comment:预览容量配置是否有效" json:"preview_capacity_valid"`
	AcceleratorUtilization  float32 `gorm:"not null;default:0;comment:加速器计算利用率" json:"accelerator_utilization"`
	AcceleratorMetricsValid bool    `gorm:"not null;default:false;comment:加速器指标是否有效" json:"accelerator_metrics_valid"`
}

// TableName 指定表名
func (EdgeNodeEngineMetrics) TableName() string {
	return "edge_node_engine_metrics"
}

// SortableFields 返回允许排序的字段列表
func (EdgeNodeEngineMetrics) SortableFields() []string {
	return []string{"created_at", "active_stream_count", "accelerator_utilization"}
}
