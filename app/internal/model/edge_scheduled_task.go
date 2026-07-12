package model

import (
	"time"

	"gorm.io/datatypes"
)

// 计划任务命令名称常量（与 C++ CommandDispatcher 对齐）
const (
	ScheduledTaskCmdStartStream    = "start_stream"
	ScheduledTaskCmdStopStream     = "stop_stream"
	ScheduledTaskCmdStartPlayback  = "start_playback"
	ScheduledTaskCmdStopPlayback   = "stop_playback"
	ScheduledTaskCmdStreamStatus   = "stream_status"
	ScheduledTaskCmdSelfCheck      = "self_check"
	ScheduledTaskCmdFaceLibrary    = "face_library"
	ScheduledTaskCmdAlgoWarmup     = "algo_warmup"
	ScheduledTaskCmdFaceEmbedding  = "face_embedding"
)

// EdgeScheduledTask 计划任务定义
type EdgeScheduledTask struct {
	BaseModel
	Name              string         `gorm:"type:varchar(255);not null;comment:任务名称" json:"name"`
	Description       string         `gorm:"type:varchar(500);comment:任务描述" json:"description"`
	CronExpr          string         `gorm:"type:varchar(100);not null;comment:cron 表达式" json:"cron_expr"`
	TargetType        string         `gorm:"type:varchar(20);not null;comment:目标类型(single_node/tag)" json:"target_type"`
	TargetID          string         `gorm:"type:uuid;not null;comment:目标ID(node_id 或 tag_id)" json:"target_id"`
	CommandName       string         `gorm:"type:varchar(100);not null;comment:命令名称" json:"command_name"`
	CommandParams     datatypes.JSON `gorm:"type:jsonb;default:'{}';comment:命令参数" json:"command_params,omitempty" swaggertype:"object"`
	WaitResponse      bool           `gorm:"default:false;comment:是否同步等待响应" json:"wait_response"`
	WaitTimeoutSec    int            `gorm:"default:30;comment:同步等待超时(秒)" json:"wait_timeout_sec"`
	Enabled           bool           `gorm:"default:true;comment:是否启用" json:"enabled"`
	MaxRetries        int            `gorm:"default:3;comment:失败后最大重试次数" json:"max_retries"`
	RetryIntervalSec  int            `gorm:"default:60;comment:重试间隔(秒)" json:"retry_interval_sec"`
	LastRunAt         *time.Time     `gorm:"comment:上次运行时间" json:"last_run_at,omitempty"`
}

// TableName 指定表名
func (EdgeScheduledTask) TableName() string {
	return "edge_scheduled_tasks"
}

// SortableFields 返回允许排序的字段列表
func (EdgeScheduledTask) SortableFields() []string {
	return []string{"created_at", "updated_at", "name", "enabled"}
}
