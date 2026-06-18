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
	Name        string `json:"name" binding:"required,min=1,max=128"`
	Description string `json:"description" binding:"max=500"`
	Endpoint    string `json:"endpoint" binding:"required,url,max=255"`
	MaxLoad     int    `json:"max_load" binding:"required,min=1"`
	Remark      string `json:"remark" binding:"max=1000"`
}

// UpdateEdgeNodeRequest 更新边缘节点请求
type UpdateEdgeNodeRequest struct {
	Name        string `json:"name" binding:"omitempty,min=1,max=128"`
	Description string `json:"description" binding:"max=500"`
	Endpoint    string `json:"endpoint" binding:"omitempty,url,max=255"`
	MaxLoad     int    `json:"max_load" binding:"omitempty,min=1"`

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
