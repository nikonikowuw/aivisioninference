package model

import "time"

// TerminalSession 状态常量
const (
	TerminalSessionStatusActive = "active"
	TerminalSessionStatusPaused = "paused"
	TerminalSessionStatusClosed = "closed"
)

// TerminalSession 终端会话模型
type TerminalSession struct {
	BaseModel

	NodeID          string     `gorm:"type:uuid;not null;index:idx_terminal_sessions_node_id;comment:边缘节点ID" json:"node_id"`
	UserID          string     `gorm:"type:uuid;not null;index:idx_terminal_sessions_user_id;comment:用户ID" json:"user_id"`
	UserName        string     `gorm:"type:varchar(100);not null;comment:用户名" json:"user_name"`
	Status          string     `gorm:"type:varchar(20);not null;default:active;comment:状态(active/paused/closed)" json:"status"`
	Reason          string     `gorm:"type:varchar(50);comment:关闭原因(closed/timeout/error)" json:"reason"`
	ErrorMessage    string     `gorm:"type:text;comment:错误信息" json:"error_message,omitempty"`
	StartedAt       time.Time  `gorm:"not null;comment:会话开始时间" json:"started_at"`
	EndedAt         *time.Time `gorm:"comment:会话结束时间" json:"ended_at,omitempty"`
	DurationSeconds int        `gorm:"comment:持续时长(秒)" json:"duration_seconds"`
	RecordingData   []byte     `gorm:"type:jsonb;comment:ttyrec录制数据(JSONB)" json:"recording_data,omitempty"`
	SessChKey       string     `gorm:"type:varchar(64);uniqueIndex:idx_terminal_sessions_sess_ch_key;comment:SSH Session通道key(连接池索引)" json:"-"`
	PausedAt        *time.Time `gorm:"comment:暂停时间" json:"paused_at,omitempty"`
	Node            *EdgeNode  `gorm:"foreignKey:NodeID" json:"node,omitempty"`
}

// TableName 指定表名
func (TerminalSession) TableName() string {
	return "terminal_sessions"
}

// SortableFields 返回允许排序的字段列表
func (TerminalSession) SortableFields() []string {
	return []string{"created_at", "updated_at", "started_at", "ended_at", "duration_seconds"}
}
