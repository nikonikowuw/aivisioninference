// Package model 定义系统数据模型，包含 GORM 结构体和数据库表映射。
package model

import (
	"time"
)

// 存储常量定义
const (
	StorageTypeLocal  = "local"
	StorageTypePgLOb  = "pg_lob"
	StorageTypeOSS    = "oss"

	SpaceTypeSnapshot  = "snapshot"
	SpaceTypeRecording = "recording"
	SpaceTypePerson    = "person"
	SpaceTypeAlgorithm = "algorithm"
	SpaceTypeSystem    = "system"

	ThresholdTypePercent       = "percent"
	ThresholdTypeFixedCapacity = "fixed_capacity"

	StorageCleanupStatusRunning = "running"
	StorageCleanupStatusSuccess = "success"
	StorageCleanupStatusFailed  = "failed"
)

// StorageSpace 表示存储空间，每个设备可拥有独立的抓拍/录像空间和保留策略
type StorageSpace struct {
	BaseModel
	SpaceName         string  `gorm:"type:varchar(128);not null" json:"space_name"`
	SpaceType         string  `gorm:"type:varchar(16);not null;check:space_type IN ('snapshot','recording','person','algorithm','system')" json:"space_type"`
	DeviceID          *string `gorm:"type:uuid;index" json:"device_id,omitempty"`
	StorageType       string  `gorm:"type:varchar(16);not null;default:local;check:storage_type IN ('local','pg_lob','oss')" json:"storage_type"`
	LocalBasePath     string  `gorm:"type:varchar(512);default:/data/storage" json:"local_base_path"`
	OSSEndpoint       string  `gorm:"type:varchar(255)" json:"oss_endpoint,omitempty"`
	OSSBucket         string  `gorm:"type:varchar(128)" json:"oss_bucket,omitempty"`
	OSSAccessKey      string  `gorm:"type:text" json:"-"`
	OSSSecretKey      string  `gorm:"type:text" json:"-"`
	OSSRegion         string  `gorm:"type:varchar(64)" json:"oss_region,omitempty"`
	SaveMode          int     `gorm:"type:smallint;default:0;check:save_mode IN (0,1)" json:"save_mode"`
	RetentionDays     int     `gorm:"default:0;check:retention_days >= 0" json:"retention_days"`
	CleanupMode       string  `gorm:"type:varchar(16);not null;default:cron;check:cleanup_mode IN ('cron','threshold')" json:"cleanup_mode"`
	ThresholdType     string  `gorm:"type:varchar(16);not null;default:percent;check:threshold_type IN ('percent','fixed_capacity')" json:"threshold_type"`
	ThresholdValue    float64 `gorm:"type:numeric(5,2);default:90.00" json:"threshold_value"`
	TargetPercentage  float64 `gorm:"type:numeric(5,2);default:70.00" json:"target_percentage"`
	CronExpression    string  `gorm:"type:varchar(64);default:'0 2 * * *'" json:"cron_expression"`
	ThresholdCheckCron string `gorm:"type:varchar(64);default:'*/5 * * * *'" json:"threshold_check_cron"`
	Enabled           bool    `gorm:"default:true" json:"enabled"`
	Description       string  `gorm:"type:varchar(255)" json:"description,omitempty"`
}

// SortableFields 返回允许排序的字段列表
func (StorageSpace) SortableFields() []string {
	return []string{"created_at", "space_name", "space_type"}
}

// TableName 覆盖 StorageSpace 的默认表名
func (StorageSpace) TableName() string {
	return "storage_spaces"
}

// StorageCleanupLog 表示存储清理记录，追踪每次清理操作的执行情况
type StorageCleanupLog struct {
	BaseModel
	SpaceID           string     `gorm:"type:uuid;not null" json:"space_id"`
	DiskUsageBefore   int64      `json:"disk_usage_before,omitempty"`
	DiskUsageAfter    int64      `json:"disk_usage_after,omitempty"`
	CleanedRecords    int        `gorm:"default:0" json:"cleaned_records"`
	CleanedFiles      int        `gorm:"default:0" json:"cleaned_files"`
	FreedSpace        int64      `gorm:"default:0" json:"freed_space"`
	DroppedPartitions []string   `gorm:"type:text[]" json:"dropped_partitions,omitempty"`
	Status            string     `gorm:"type:varchar(16);not null;default:success" json:"status"`
	ErrorMessage      string     `gorm:"type:text" json:"error_message,omitempty"`
	StartedAt         time.Time  `json:"started_at"`
	CompletedAt       *time.Time `json:"completed_at,omitempty"`
}

// SortableFields 返回允许排序的字段列表
func (StorageCleanupLog) SortableFields() []string {
	return []string{"created_at", "started_at", "status"}
}
