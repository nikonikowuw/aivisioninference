// Package model 定义系统数据模型，包含 GORM 结构体和数据库表映射。
package model

import (
	"time"

	"gorm.io/datatypes"
)

// 智能记录常量定义
const (
	RecordTypeRecognition = "recognition"
	RecordTypeAlarm       = "alarm"
	RecordTypeCapture     = "capture"
)

// SmartRecord 表示智能记录（识别/告警/抓拍），按 capture_time 日分区
// 注意：GORM AutoMigrate 会创建母表，但分区子表需要通过手动 SQL 创建
// 分区策略：RANGE PARTITION BY capture_time
type SmartRecord struct {
	RecordID           string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"record_id"`
	RecordType         string         `gorm:"type:varchar(16);not null;check:record_type IN ('recognition','alarm','capture');index" json:"record_type"`
	CaptureTime        time.Time      `gorm:"primaryKey;not null;index" json:"capture_time"`
	TaskID             *string        `gorm:"type:uuid" json:"task_id,omitempty"`
	TaskName           string         `gorm:"type:varchar(128)" json:"task_name,omitempty"`
	DeviceID           *string        `gorm:"type:uuid;index" json:"device_id,omitempty"`
	DeviceName         string         `gorm:"type:varchar(128)" json:"device_name,omitempty"`
	AlgorithmName      string         `gorm:"type:varchar(128)" json:"algorithm_name,omitempty"`
	AlgorithmVersion   string         `gorm:"type:varchar(32)" json:"algorithm_version,omitempty"`
	CategoryCode       *int           `gorm:"index" json:"category_code,omitempty"`
	Confidence         *float64       `gorm:"type:numeric(5,4);check:confidence >= 0 AND confidence <= 1" json:"confidence,omitempty"`
	PersonRecordID     *string        `gorm:"type:uuid" json:"person_record_id,omitempty"`
	PersonName         string         `gorm:"type:varchar(128)" json:"person_name,omitempty"`
	Similarity         *float64       `gorm:"type:numeric(5,4);check:similarity >= 0 AND similarity <= 1" json:"similarity,omitempty"`
	IdentityID         string         `gorm:"type:varchar(64)" json:"identity_id,omitempty"`
	TriggeredLineIDs   []string       `gorm:"type:text[]" json:"triggered_line_ids,omitempty"`
	Direction          string         `gorm:"type:varchar(32)" json:"direction,omitempty"`
	AlarmType          string         `gorm:"type:varchar(64);index" json:"alarm_type,omitempty"`
	AlarmLevel         string         `gorm:"type:varchar(16)" json:"alarm_level,omitempty"`
	AlarmMajor         string         `gorm:"type:varchar(128)" json:"alarm_major,omitempty"`
	CorrelationID      *string        `gorm:"type:uuid;index" json:"correlation_id,omitempty"`
	BusinessTags       []string       `gorm:"type:text[]" json:"business_tags,omitempty"`
	SnapshotImageURL   string         `gorm:"type:varchar(512)" json:"snapshot_image_url,omitempty"`
	TargetCropURL      string         `gorm:"type:varchar(512)" json:"target_crop_url,omitempty"`
	BackgroundImageURL string         `gorm:"type:varchar(512)" json:"background_image_url,omitempty"`
	RawResult          datatypes.JSON `gorm:"type:jsonb" json:"raw_result,omitempty"`
	CreatedAt          time.Time      `json:"created_at"`
}

// TableName 覆盖 SmartRecord 的默认表名
func (SmartRecord) TableName() string {
	return "smart_records"
}

// SortableFields 返回允许排序的字段列表
func (SmartRecord) SortableFields() []string {
	return []string{"capture_time", "created_at", "record_type"}
}
