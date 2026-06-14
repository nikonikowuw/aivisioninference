package model

import (
	"gorm.io/datatypes"
)

// AIVisionTask 状态常量
const (
	TaskStatusDraft    = "draft"
	TaskStatusReady    = "ready"
	TaskStatusRunning  = "running"
	TaskStatusError    = "error"
	TaskStatusStopped  = "stopped"
	TaskStatusSuspended = "suspended"
)

// AIVisionTask AI视觉推理任务模型
type AIVisionTask struct {
	BaseModel
	Name            string         `gorm:"type:varchar(255);not null;comment:任务名称" json:"name"`
	Status          string         `gorm:"type:varchar(50);not null;default:'draft';comment:状态(draft/ready/running/error)" json:"status"`
	ScheduleID      string         `gorm:"type:uuid;not null;comment:关联时间配置ID" json:"schedule_id"`
	DeviceChannelID string         `gorm:"type:uuid;not null;comment:关联GB28181通道ID" json:"device_channel_id"`
	AlgoPackageID   string         `gorm:"type:uuid;not null;comment:关联算法包ID" json:"algo_package_id"`
	TargetNodeID    string         `gorm:"type:uuid;not null;comment:关联边缘节点ID" json:"target_node_id"`
	StartDate       datatypes.Date `gorm:"type:date;not null;comment:生效起始日期(来自时间配置)" json:"start_date" swaggertype:"string"`
	EndDate         datatypes.Date `gorm:"type:date;not null;comment:生效结束日期(来自时间配置)" json:"end_date" swaggertype:"string"`
	TimeWindows     datatypes.JSON `gorm:"type:jsonb;not null;comment:每天运行的时间段(来自时间配置)" json:"time_windows" swaggertype:"object"`
	AIParams        datatypes.JSON `gorm:"type:jsonb" json:"ai_params,omitempty" swaggertype:"object"`
	ROIRegions      datatypes.JSON `gorm:"type:jsonb;default:'[]'" json:"roi_regions" swaggertype:"object"`
	MarkRegions     datatypes.JSON `gorm:"type:jsonb;default:'[]'" json:"mark_regions" swaggertype:"object"`
	LineRegions     datatypes.JSON `gorm:"type:jsonb;default:'[]'" json:"line_regions" swaggertype:"object"`
	ErrorReason     string         `gorm:"type:varchar(500);comment:异常停止原因" json:"error_reason"`
}

// TableName 指定表名
func (AIVisionTask) TableName() string {
	return "ai_vision_tasks"
}

// SortableFields 返回允许排序的字段列表
func (AIVisionTask) SortableFields() []string {
	return []string{"created_at", "updated_at"}
}
