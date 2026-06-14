package model

import (
	"time"
)

// EdgeNodeAlgorithm 状态常量
const (
	AlgoDeployPending     = "pending"
	AlgoDeployDownloading = "downloading"
	AlgoDeployInstalled   = "installed"
	AlgoDeployFailed      = "failed"
)

// EdgeNodeAlgorithm 边缘节点-算法关联模型
type EdgeNodeAlgorithm struct {
	BaseModel
	NodeID        string            `gorm:"type:uuid;not null;uniqueIndex:idx_node_algo;comment:关联边缘节点ID" json:"node_id"`
	AlgoPackageID string            `gorm:"type:uuid;not null;uniqueIndex:idx_node_algo;comment:关联算法包ID" json:"algo_package_id"`
	Status        string            `gorm:"type:varchar(50);not null;default:'pending';comment:安装状态(pending/downloading/installed/failed)" json:"status"`
	InstallPath   string            `gorm:"type:varchar(500);comment:安装路径" json:"install_path"`
	DeployedAt    *time.Time        `json:"deployed_at,omitempty"`
	ErrorMessage  string            `gorm:"type:varchar(1000);comment:错误消息" json:"error_message"`
	RetryCount    int               `gorm:"default:0;comment:重试次数" json:"retry_count"`
	LastRetryAt   *time.Time        `json:"last_retry_at,omitempty"`

	// Relations
	Node        *EdgeNode         `gorm:"foreignKey:NodeID;references:ID;constraint:OnDelete:CASCADE" json:"node,omitempty"`
	AlgoPackage *AlgorithmPackage `gorm:"foreignKey:AlgoPackageID;references:ID;constraint:OnDelete:CASCADE" json:"algo_package,omitempty"`
}

// TableName 指定表名
func (EdgeNodeAlgorithm) TableName() string {
	return "edge_node_algorithms"
}

// SortableFields 返回允许排序的字段列表
func (EdgeNodeAlgorithm) SortableFields() []string {
	return []string{"created_at", "updated_at", "deployed_at"}
}
