// AIVisionTask DTO 定义
package dto

import (
	"encoding/json"

	"github.com/niko-admin/niko-admin/internal/pkg/scopes"
)

// TimeWindow 时间段（保留用于冲突检查等场景）
type TimeWindow struct {
	Start string `json:"start" binding:"required"`
	End   string `json:"end" binding:"required"`
}

// CheckConflictRequest 资源冲突检查请求
type CheckConflictRequest struct {
	TargetNodeID  string       `json:"target_node_id" binding:"required,uuid"`
	StartDate     string       `json:"start_date" binding:"required"`
	EndDate       string       `json:"end_date" binding:"required"`
	TimeWindows   []TimeWindow `json:"time_windows" binding:"required,min=1"`
	ExcludeTaskID string       `json:"exclude_task_id,omitempty"`
}

// CreateAIVisionTaskRequest 创建推理任务请求
// 通过 schedule_id 引用时间配置，不再在任务上直接设置时间窗。
type CreateAIVisionTaskRequest struct {
	Name            string          `json:"name" binding:"required,max=255"`
	ScheduleID      string          `json:"schedule_id" binding:"required,uuid"`
	DeviceChannelID string          `json:"device_channel_id" binding:"required,uuid"`
	AlgoPackageID   string          `json:"algo_package_id" binding:"required,uuid"`
	TargetNodeID    string          `json:"target_node_id" binding:"required,uuid"`
	AIParams        json.RawMessage `json:"ai_params,omitempty" swaggertype:"object"`
	ROIRegions      json.RawMessage `json:"roi_regions,omitempty" swaggertype:"object"`
	MarkRegions     json.RawMessage `json:"mark_regions,omitempty" swaggertype:"object"`
	LineRegions     json.RawMessage `json:"line_regions,omitempty" swaggertype:"object"`
}

// UpdateAIVisionTaskRequest 更新推理任务请求
type UpdateAIVisionTaskRequest struct {
	Name            string          `json:"name" binding:"required,max=255"`
	Status          string          `json:"status"`
	ScheduleID      string          `json:"schedule_id" binding:"required,uuid"`
	DeviceChannelID string          `json:"device_channel_id" binding:"required,uuid"`
	AlgoPackageID   string          `json:"algo_package_id" binding:"required,uuid"`
	TargetNodeID    string          `json:"target_node_id" binding:"required,uuid"`
	AIParams        json.RawMessage `json:"ai_params,omitempty" swaggertype:"object"`
	ROIRegions      json.RawMessage `json:"roi_regions,omitempty" swaggertype:"object"`
	MarkRegions     json.RawMessage `json:"mark_regions,omitempty" swaggertype:"object"`
	LineRegions     json.RawMessage `json:"line_regions,omitempty" swaggertype:"object"`
	ErrorReason     string          `json:"error_reason"`
}

// AIVisionTaskListRequest 推理任务分页列表请求
type AIVisionTaskListRequest struct {
	PageRequest
	Keyword string `form:"keyword"`
}

// FilterScopes 构建筛选条件
func (r *AIVisionTaskListRequest) FilterScopes() []scopes.Scope {
	var sc []scopes.Scope
	if r.Keyword != "" {
		sc = append(sc, scopes.MultiLike([]string{"name", "status", "device_channel_id", "algo_package_id", "target_node_id", "error_reason"}, r.Keyword))
	}
	return sc
}
