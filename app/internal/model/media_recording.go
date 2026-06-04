package model

import "time"

// MediaRecording represents a recorded video file from ZLMediaKit.
type MediaRecording struct {
	BaseModel
	DeviceID      string     `gorm:"type:uuid;not null;index" json:"device_id"`
	App           string     `gorm:"type:varchar(64);not null" json:"app"`
	Stream        string     `gorm:"type:varchar(128);not null" json:"stream"`
	FileName      string     `gorm:"type:varchar(255);not null" json:"file_name"`
	FilePath      string     `gorm:"type:text;not null" json:"file_path"`
	FileSize      int64      `json:"file_size"`
	DurationSec   int        `json:"duration_sec"`
	StartTime     time.Time  `json:"start_time"`
	EndTime       time.Time  `json:"end_time"`
	RecordType    string     `gorm:"type:varchar(16);default:manual" json:"record_type"` // manual, plan, alarm
	Status        string     `gorm:"type:varchar(16);default:completed" json:"status"`
}

// SortableFields returns the list of sortable fields.
func (MediaRecording) SortableFields() []string {
	return []string{"created_at", "start_time", "end_time", "file_size"}
}

// TableName overrides the table name.
func (MediaRecording) TableName() string {
	return "media_recordings"
}
