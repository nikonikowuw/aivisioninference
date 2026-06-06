// Package model 定义系统数据模型，包含 GORM 结构体和数据库表映射。
package model

import (
	"time"

	"github.com/pgvector/pgvector-go"
)

// 人员常量定义
const (
	GenderUnknown = "unknown"
	GenderMale    = "male"
	GenderFemale  = "female"
	GenderOther   = "other"

	EmbeddingStatusPending    = "pending"
	EmbeddingStatusExtracting = "extracting"
	EmbeddingStatusActive     = "active"
	EmbeddingStatusFailed     = "failed"
	EmbeddingStatusDisabled   = "disabled"

	ImportTaskStatusPending   = "pending"
	ImportTaskStatusRunning   = "running"
	ImportTaskStatusCompleted = "completed"
	ImportTaskStatusFailed    = "failed"

	ImportTaskTypePerson = "person"
	ImportTaskTypeDevice = "device"
)

// PersonGroup 表示人员分组，支持单层自关联分组
type PersonGroup struct {
	BaseModel
	GroupName   string         `gorm:"type:varchar(128);not null" json:"group_name"`
	Description string         `gorm:"type:varchar(255)" json:"description"`
	ParentID    *string        `gorm:"type:uuid" json:"parent_id"`
	Parent      *PersonGroup   `gorm:"foreignKey:ParentID" json:"parent,omitempty"`
	Children    []PersonGroup  `gorm:"foreignKey:ParentID" json:"children,omitempty"`
	SortOrder   int            `gorm:"default:0" json:"sort_order"`
}

// SortableFields 返回允许排序的字段列表
func (PersonGroup) SortableFields() []string {
	return []string{"created_at", "sort_order", "group_name"}
}

// Person 表示人员记录，遵循"一张人脸图对应一条记录"原则
type Person struct {
	BaseModel
	PersonCode              string          `gorm:"type:varchar(64);index" json:"person_code"`
	PersonName              string          `gorm:"type:varchar(128);not null;index" json:"person_name"`
	Gender                  string          `gorm:"type:varchar(16);default:unknown;check:gender IN ('unknown','male','female','other')" json:"gender"`
	Phone                   string          `gorm:"type:varchar(32)" json:"phone"`
	IDNumber                string          `gorm:"type:varchar(64)" json:"id_number,omitempty"`
	ImageURL                string          `gorm:"type:varchar(512);not null" json:"image_url"`
	ImageMD5                string          `gorm:"type:varchar(64);uniqueIndex" json:"image_md5"`
	FaceQualityScore        *float64        `gorm:"type:numeric(4,3);check:face_quality_score >= 0 AND face_quality_score <= 1" json:"face_quality_score,omitempty"`
	EmbeddingStatus         string          `gorm:"type:varchar(16);not null;default:pending;index" json:"embedding_status"`
	EmbeddingErrorCode      string          `gorm:"type:varchar(64)" json:"embedding_error_code,omitempty"`
	EmbeddingErrorMessageKey string         `gorm:"type:varchar(128)" json:"embedding_error_message_key,omitempty"`
	EmbeddingRetryable      bool            `gorm:"default:false" json:"embedding_retryable"`
	Enabled                 bool            `gorm:"default:true;index" json:"enabled"`
	Remark                  string          `gorm:"type:text" json:"remark"`
	Groups                  []PersonGroup   `gorm:"many2many:person_group_members;foreignKey:ID;joinForeignKey:PersonRecordID;references:ID" json:"groups,omitempty"`
	Embedding               *PersonEmbedding `gorm:"foreignKey:PersonRecordID" json:"embedding,omitempty"`
}

// SortableFields 返回允许排序的字段列表
func (Person) SortableFields() []string {
	return []string{"created_at", "updated_at", "person_name", "embedding_status"}
}

// TableName 覆盖 Person 的默认表名
func (Person) TableName() string {
	return "persons"
}

// PersonGroupMember 是人员与分组多对多关系的关联表
type PersonGroupMember struct {
	PersonRecordID string    `gorm:"type:uuid;primaryKey" json:"person_record_id"`
	GroupID        string    `gorm:"type:uuid;primaryKey" json:"group_id"`
	CreatedAt      time.Time `json:"created_at"`
}

// PersonEmbedding 表示人脸特征向量，使用 pgvector 512 维
type PersonEmbedding struct {
	BaseModel
	PersonRecordID string          `gorm:"type:uuid;uniqueIndex;not null" json:"person_record_id"`
	Embedding      pgvector.Vector `gorm:"type:vector(512);not null" json:"-"`
	Version        int             `gorm:"default:1" json:"version"`
}

// SortableFields 返回允许排序的字段列表
func (PersonEmbedding) SortableFields() []string {
	return []string{"created_at", "version"}
}

// ImportTask 表示批量导入任务，追踪导入进度和结果
type ImportTask struct {
	BaseModel
	TaskType      string     `gorm:"type:varchar(16);not null;index;check:task_type IN ('person','device')" json:"task_type"`
	FileName      string     `gorm:"type:varchar(255);not null" json:"file_name"`
	FileURL       string     `gorm:"type:varchar(512)" json:"file_url"`
	TotalRows     int        `gorm:"default:0" json:"total_rows"`
	SuccessRows   int        `gorm:"default:0" json:"success_rows"`
	FailedRows    int        `gorm:"default:0" json:"failed_rows"`
	FailDetailURL string     `gorm:"type:varchar(512)" json:"fail_detail_url"`
	Status        string     `gorm:"type:varchar(16);not null;default:pending;index" json:"status"`
	FileMD5       string     `gorm:"type:varchar(64)" json:"file_md5,omitempty"`
	ExternalKey   string     `gorm:"type:varchar(128);uniqueIndex" json:"external_key,omitempty"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
}

// SortableFields 返回允许排序的字段列表
func (ImportTask) SortableFields() []string {
	return []string{"created_at", "task_type", "status"}
}


