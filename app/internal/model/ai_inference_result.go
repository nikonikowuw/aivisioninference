package model

import (
	"time"

	"gorm.io/datatypes"
)

// AIInferenceResult 表示 AI 推理结果记录
type AIInferenceResult struct {
	BaseModel
	DeviceID    string         `gorm:"type:uuid;not null;index" json:"device_id"`
	SnapshotPath string        `gorm:"type:text" json:"snapshot_path"`
	Algorithms  string         `gorm:"type:varchar(255)" json:"algorithms"`
	Result      datatypes.JSON `gorm:"type:jsonb" json:"result"`
	Confidence  float64        `json:"confidence"`
	AlertType   string         `gorm:"type:varchar(64)" json:"alert_type,omitempty"`
	TaskType    string         `gorm:"type:varchar(20)" json:"task_type"`
	ProcessedAt time.Time      `json:"processed_at"`
}

// SortableFields returns the list of sortable fields.
func (AIInferenceResult) SortableFields() []string {
	return []string{"created_at", "processed_at", "confidence"}
}

// TableName overrides the table name.
func (AIInferenceResult) TableName() string {
	return "ai_inference_results"
}
