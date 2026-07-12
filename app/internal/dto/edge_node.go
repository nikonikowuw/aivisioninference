package dto

import (
	"github.com/niko-admin/niko-admin/internal/pkg/scopes"
)

// EdgeNodeListRequest 边缘节点列表查询参数
type EdgeNodeListRequest struct {
	PageRequest
	Keyword string `form:"keyword"`
	Status  string `form:"status" binding:"omitempty,oneof=online offline error disabled"`
}

// FilterScopes 返回当前请求对应的 GORM 查询范围函数列表
func (r *EdgeNodeListRequest) FilterScopes() []scopes.Scope {
	var sc []scopes.Scope
	if r.Keyword != "" {
		sc = append(sc, scopes.MultiLike([]string{"name", "description", "endpoint", "hal_platform"}, r.Keyword))
	}
	if r.Status != "" {
		sc = append(sc, scopes.Eq("status", r.Status))
	}
	return sc
}

// CreateEdgeNodeRequest 创建边缘节点请求
type CreateEdgeNodeRequest struct {
	Name                   string `json:"name" binding:"required,min=1,max=128"`
	Description            string `json:"description" binding:"max=500"`
	Endpoint               string `json:"endpoint" binding:"required,url,max=255"`
	MaxLoad                int    `json:"max_load" binding:"required,min=1"`
	MediaDecodeCapacity    int    `json:"media_decode_capacity" binding:"min=0"`
	MediaEncodeCapacity    int    `json:"media_encode_capacity" binding:"min=0"`
	MediaEgressCapacityBPS int64  `json:"media_egress_capacity_bps" binding:"min=0"`
	MediaMetricsTTLSeconds int    `json:"media_metrics_ttl_seconds" binding:"min=0"`
	Remark                 string `json:"remark" binding:"max=1000"`
}

// UpdateEdgeNodeRequest 更新边缘节点请求
type UpdateEdgeNodeRequest struct {
	Name                   string `json:"name" binding:"omitempty,min=1,max=128"`
	Description            string `json:"description" binding:"max=500"`
	Endpoint               string `json:"endpoint" binding:"omitempty,url,max=255"`
	MaxLoad                int    `json:"max_load" binding:"omitempty,min=1"`
	MediaDecodeCapacity    *int   `json:"media_decode_capacity" binding:"omitempty,min=0"`
	MediaEncodeCapacity    *int   `json:"media_encode_capacity" binding:"omitempty,min=0"`
	MediaEgressCapacityBPS *int64 `json:"media_egress_capacity_bps" binding:"omitempty,min=0"`
	MediaMetricsTTLSeconds *int   `json:"media_metrics_ttl_seconds" binding:"omitempty,min=0"`

	Enabled *bool  `json:"enabled"`
	Remark  string `json:"remark" binding:"max=1000"`
	Status  string `json:"status" binding:"omitempty,oneof=online offline error disabled"`
}

// HardwareInfo 硬件信息
type HardwareInfo struct {
	CPUModel    string `json:"cpu_model"`
	GPUModel    string `json:"gpu_model"`
	TotalMemory int64  `json:"total_memory" binding:"min=0"`
	CPUCores    int    `json:"cpu_cores" binding:"min=0"`
}

// InstalledAlgorithmInfo 已安装算法信息
type InstalledAlgorithmInfo struct {
	AlgoPackageID       string `json:"algo_package_id" binding:"required,uuid"`
	AlgoName            string `json:"algo_name,omitempty"`
	Version             string `json:"version" binding:"required"`
	InstallPath         string `json:"install_path" binding:"required"`
	Status              string `json:"status" binding:"required"`
	RuntimeStatus       string `json:"runtime_status,omitempty"`
	SupportsEmbedding   bool   `json:"supports_embedding,omitempty"`
	SupportsFaceLibrary bool   `json:"supports_face_library,omitempty"`
	EmbeddingCapacity   int    `json:"embedding_capacity,omitempty" binding:"min=0"`
}

// DiskInfo 磁盘挂载点信息
type DiskInfo struct {
	Path         string  `json:"path"`
	TotalBytes   uint64  `json:"total_bytes"`
	UsedBytes    uint64  `json:"used_bytes"`
	UsagePercent float64 `json:"usage_percent"`
}

// HeartbeatRequest 边缘节点心跳上报请求
type HeartbeatRequest struct {
	Uptime              int64                    `json:"uptime" binding:"min=0"`
	CurrentLoad         int                      `json:"current_load" binding:"min=0"`
	CPUUsage            float64                  `json:"cpu_usage" binding:"min=0,max=100"`
	MemoryUsage         float64                  `json:"memory_usage" binding:"min=0,max=100"`
	EngineVersion       string                   `json:"engine_version" binding:"required"`
	HALPlatform         string                   `json:"hal_platform" binding:"required"`
	HardwareInfo        HardwareInfo             `json:"hardware_info" binding:"required"`
	InstalledAlgorithms []InstalledAlgorithmInfo `json:"installed_algorithms"`
	ActiveStreams       []string                 `json:"active_streams"`
	Status              string                   `json:"status" binding:"omitempty"`
	ErrorMessage        string                   `json:"error_message" binding:"omitempty"`

	// Extended metrics (from C++ Engine MetricsFlattener)
	CPULoad1m    float64    `json:"cpu_load_1m,omitempty"`
	CPULoad5m    float64    `json:"cpu_load_5m,omitempty"`
	CPULoad15m   float64    `json:"cpu_load_15m,omitempty"`
	Disks        []DiskInfo `json:"disks,omitempty"`
	NetRXBytes   int64      `json:"net_rx_bytes,omitempty"`
	NetTXBytes   int64      `json:"net_tx_bytes,omitempty"`
	NetRXSpeed   float64    `json:"net_rx_speed,omitempty"`
	NetTXSpeed   float64    `json:"net_tx_speed,omitempty"`
	ProcessCount int32      `json:"process_count,omitempty"`
	ThreadCount  int32      `json:"thread_count,omitempty"`
	Temperature  float64    `json:"temperature,omitempty"`
}

// PendingDeployment 待下发算法包信息
type PendingDeployment struct {
	AlgoPackageID string `json:"algo_package_id"`
	DownloadURL   string `json:"download_url"`
	MD5           string `json:"md5"`
	ExtractPath   string `json:"extract_path"`
	AlgoName      string `json:"algo_name"`
	Version       string `json:"version"`
}

// HeartbeatResponse 心跳上报响应
type HeartbeatResponse struct {
	PendingDeployments []PendingDeployment `json:"pending_deployments"`
}

// DeployAlgorithmRequest 部署算法请求
type DeployAlgorithmRequest struct {
	AlgoPackageID string `json:"algo_package_id" binding:"required,uuid"`
}

// DeployAlgorithmResponse 部署算法响应
type DeployAlgorithmResponse struct {
	DeploymentID string `json:"deployment_id"`
	Message      string `json:"message"`
}
