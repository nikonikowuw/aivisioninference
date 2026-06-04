// Package model 定义系统数据模型，包含 GORM 结构体和数据库表映射。
package model

import (
	"time"

	"gorm.io/datatypes"
)

// 推理任务常量定义
const (
	InferTaskStatusPending      = "pending"
	InferTaskStatusRunning      = "running"
	InferTaskStatusReconnecting = "reconnecting"
	InferTaskStatusStopped      = "stopped"
	InferTaskStatusFailed       = "failed"

	InferTaskAlgoStatusPending = "pending"
	InferTaskAlgoStatusRunning = "running"
	InferTaskAlgoStatusStopped = "stopped"
	InferTaskAlgoStatusFailed  = "failed"

	// 解码硬件类型
	DecodeHWTypeAuto   = 0
	DecodeHWTypeRKMPP  = 1
	DecodeHWTypeDVPP   = 2
	DecodeHWTypeFFmpeg = 3
)

// InferTask 表示推理任务，每个设备只能有一个激活的推理任务（一机一任务）
// 注意：与 scaffold model.Task（后台异步任务）不同，InferTask 管理视频流推理生命周期
type InferTask struct {
	BaseModel
	TaskName          string     `gorm:"type:varchar(128);not null" json:"task_name"`
	DeviceID          string     `gorm:"type:uuid;uniqueIndex:idx_infer_task_device_unique;not null" json:"device_id"`
	StreamURL         string     `gorm:"type:text;not null" json:"stream_url"`
	DecodeHWType      int        `gorm:"type:int;default:0;check:decode_hw_type IN (0,1,2,3)" json:"decode_hw_type"`
	Status            string     `gorm:"type:varchar(16);not null;default:pending;index" json:"status"`
	ErrorCode         string     `gorm:"type:varchar(64)" json:"error_code,omitempty"`
	ErrorMessage      string     `gorm:"type:text" json:"error_message,omitempty"`
	ReconnectCount    int        `gorm:"default:0" json:"reconnect_count"`
	MaxReconnects     int        `gorm:"default:3" json:"max_reconnects"`
	ReconnectInterval int        `gorm:"default:5" json:"reconnect_interval"`
	StartedAt         *time.Time `json:"started_at,omitempty"`
	StoppedAt         *time.Time `json:"stopped_at,omitempty"`
	Version           int        `gorm:"default:1" json:"version"`
	Remark            string     `gorm:"type:text" json:"remark,omitempty"`
	Algorithms        []InferTaskAlgorithm `gorm:"foreignKey:TaskID" json:"algorithms,omitempty"`
}

// SortableFields 返回允许排序的字段列表
func (InferTask) SortableFields() []string {
	return []string{"created_at", "updated_at", "task_name", "status"}
}

// TableName 覆盖 InferTask 的默认表名（使用 infer_tasks 避免与 scaffold model.Task 冲突）
func (InferTask) TableName() string {
	return "infer_tasks"
}

// InferTaskAlgorithm 表示推理任务与算法的绑定配置
// 一个推理任务可配置多个算法，每个算法可独立配置参数、ROI/MARK/LINE 和生效时间段
type InferTaskAlgorithm struct {
	BaseModel
	TaskID         string         `gorm:"type:uuid;uniqueIndex:idx_infer_task_algo_unique;not null" json:"task_id"`
	PackageID      string         `gorm:"type:uuid;not null;index" json:"package_id"`
	AlgoName       string         `gorm:"type:varchar(128);uniqueIndex:idx_infer_task_algo_unique;not null" json:"algo_name"`
	AlgoVersion    string         `gorm:"type:varchar(32);not null" json:"algo_version"`
	SortOrder      int            `gorm:"default:0" json:"sort_order"`
	Status         string         `gorm:"type:varchar(16);not null;default:pending" json:"status"`
	AIParams       datatypes.JSON `gorm:"type:jsonb" json:"ai_params,omitempty"`
	ROIRegions     datatypes.JSON `gorm:"type:jsonb;default:'[]'" json:"roi_regions"`
	MarkRegions    datatypes.JSON `gorm:"type:jsonb;default:'[]'" json:"mark_regions"`
	LineRegions    datatypes.JSON `gorm:"type:jsonb;default:'[]'" json:"line_regions"`
	ScheduleEnabled bool          `gorm:"default:false" json:"schedule_enabled"`
	ScheduleStart  string         `gorm:"type:time" json:"schedule_start,omitempty"`
	ScheduleEnd    string         `gorm:"type:time" json:"schedule_end,omitempty"`
}

// SortableFields 返回允许排序的字段列表
func (InferTaskAlgorithm) SortableFields() []string {
	return []string{"created_at", "sort_order", "algo_name"}
}

// InferTaskStatusHistory 表示推理任务状态变更历史，用于运维排障
type InferTaskStatusHistory struct {
	HistoryID  string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"history_id"`
	TaskID     string    `gorm:"type:uuid;index;not null" json:"task_id"`
	FromStatus string    `gorm:"type:varchar(16)" json:"from_status,omitempty"`
	ToStatus   string    `gorm:"type:varchar(16);not null" json:"to_status"`
	Reason     string    `gorm:"type:varchar(255)" json:"reason,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

// SortableFields 返回允许排序的字段列表
func (InferTaskStatusHistory) SortableFields() []string {
	return []string{"created_at"}
}
