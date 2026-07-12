package model

import (
	"time"

	"gorm.io/gorm"
)

// EdgeNode 状态常量
const (
	NodeStatusOnline   = "online"
	NodeStatusOffline  = "offline"
	NodeStatusError    = "error"
	NodeStatusDisabled = "disabled"
)

// EdgeNode 边缘推理节点模型
type EdgeNode struct {
	BaseModel
	Name      string         `gorm:"type:varchar(255);not null;comment:节点名称" json:"name"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"` // 由部分唯一索引 WHERE deleted_at IS NULL 约束

	Description            string     `gorm:"type:varchar(500);comment:描述" json:"description"`
	Endpoint               string     `gorm:"type:varchar(255);not null;comment:HTTP访问地址" json:"endpoint"`
	AuthToken              string     `gorm:"type:text;not null;comment:JWT认证Token" json:"-"` // Never expose in JSON responses
	Status                 string     `gorm:"type:varchar(50);not null;default:'offline';comment:状态(online/offline/error/disabled)" json:"status"`
	LastHeartbeat          *time.Time `gorm:"index;comment:最后心跳时间" json:"last_heartbeat,omitempty"`
	CPUModel               string     `gorm:"type:varchar(255);comment:CPU型号" json:"cpu_model"`
	GPUModel               string     `gorm:"type:varchar(255);comment:GPU型号" json:"gpu_model"`
	HALPlatform            string     `gorm:"type:varchar(100);comment:HAL平台(rkmpp/macos/ascend)" json:"hal_platform"`
	TotalMemory            int64      `gorm:"comment:总内存(字节)" json:"total_memory"`
	CurrentLoad            int        `gorm:"default:0;comment:当前负载(任务数)" json:"current_load"`
	MaxLoad                int        `gorm:"default:1;comment:最大负载" json:"max_load"`
	EmbeddingCapacity      int        `gorm:"default:1;comment:embedding并发容量" json:"embedding_capacity"`
	MediaDecodeCapacity    int        `gorm:"not null;default:0;comment:媒体解码槽容量，0表示未配置" json:"media_decode_capacity"`
	MediaEncodeCapacity    int        `gorm:"not null;default:0;comment:媒体编码槽容量，0表示未配置" json:"media_encode_capacity"`
	MediaEgressCapacityBPS int64      `gorm:"not null;default:0;comment:媒体出口带宽上限(bit/s)，0表示未配置" json:"media_egress_capacity_bps"`
	MediaMetricsTTLSeconds int        `gorm:"not null;default:0;comment:媒体指标有效期(秒)，0表示未配置" json:"media_metrics_ttl_seconds"`
	EngineVersion          string     `gorm:"type:varchar(100);comment:引擎版本" json:"engine_version"`
	Uptime                 int64      `gorm:"comment:运行时长(秒)" json:"uptime"`
	Enabled                bool         `gorm:"default:true;comment:是否启用" json:"enabled"`
	Remark                 string       `gorm:"type:varchar(1000);comment:备注" json:"remark"`
	Tags                   []EdgeNodeTag `gorm:"many2many:edge_node_tag_relations;" json:"tags,omitempty"`
}

// TableName 指定表名
func (EdgeNode) TableName() string {
	return "edge_nodes"
}

// SortableFields 返回允许排序的字段列表
func (EdgeNode) SortableFields() []string {
	return []string{"created_at", "updated_at", "last_heartbeat"}
}
