// Package dto 定义人员管理模块请求与响应 DTO。
package dto

import (
	"strings"

	"github.com/niko-admin/niko-admin/internal/pkg/scopes"
)

// PersonListRequest 人员列表查询参数。
type PersonListRequest struct {
	PageRequest
	Keyword         string `form:"keyword" json:"keyword"`
	GroupID         string `form:"group_id" json:"group_id"`
	EmbeddingStatus string `form:"embedding_status" json:"embedding_status"`
	Enabled         *bool  `form:"enabled" json:"enabled"`
	StartTime       string `form:"start_time" json:"start_time"`
	EndTime         string `form:"end_time" json:"end_time"`
}

// FilterScopes 返回人员列表基础过滤条件。
func (r *PersonListRequest) FilterScopes() []scopes.Scope {
	var sc []scopes.Scope
	if strings.TrimSpace(r.Keyword) != "" {
		sc = append(sc, scopes.MultiLike([]string{"person_name", "person_code", "phone"}, r.Keyword))
	}
	if r.EmbeddingStatus != "" {
		sc = append(sc, scopes.Eq("embedding_status", r.EmbeddingStatus))
	}
	if r.Enabled != nil {
		sc = append(sc, scopes.Eq("enabled", *r.Enabled))
	}
	return sc
}

// PersonCreateRequest 创建人员请求。
type PersonCreateRequest struct {
	PersonCode string   `json:"person_code" form:"person_code" binding:"omitempty,max=64"`
	PersonName string   `json:"person_name" form:"person_name" binding:"required,min=1,max=128"`
	Gender     string   `json:"gender" form:"gender" binding:"omitempty,oneof=unknown male female other"`
	Phone      string   `json:"phone" form:"phone" binding:"omitempty,max=32"`
	IDNumber   string   `json:"id_number" form:"id_number" binding:"omitempty,max=64"`
	GroupIDs   []string `json:"group_ids" form:"group_ids"`
	Enabled    *bool    `json:"enabled" form:"enabled"`
	Remark     string   `json:"remark" form:"remark" binding:"omitempty,max=1000"`
	// ImageURL 通过分片上传后获得的图片存储路径，优先于 multipart 文件上传。
	ImageURL string `json:"image_url" form:"image_url" binding:"omitempty"`
}

// PersonUpdateRequest 更新人员请求。
type PersonUpdateRequest struct {
	PersonCode string   `json:"person_code" form:"person_code" binding:"omitempty,max=64"`
	PersonName string   `json:"person_name" form:"person_name" binding:"omitempty,min=1,max=128"`
	Gender     string   `json:"gender" form:"gender" binding:"omitempty,oneof=unknown male female other"`
	Phone      string   `json:"phone" form:"phone" binding:"omitempty,max=32"`
	IDNumber   string   `json:"id_number" form:"id_number" binding:"omitempty,max=64"`
	GroupIDs   []string `json:"group_ids" form:"group_ids"`
	Enabled    *bool    `json:"enabled" form:"enabled"`
	Remark     string   `json:"remark" form:"remark" binding:"omitempty,max=1000"`
	// ImageURL 通过分片上传后获得的图片存储路径，优先于 multipart 文件上传。
	ImageURL string `json:"image_url" form:"image_url" binding:"omitempty"`
}

// PersonToggleRequest 批量启用/禁用请求。
type PersonToggleRequest struct {
	IDs     []string `json:"ids" binding:"required,min=1,max=100,dive,required"`
	Enabled bool     `json:"enabled"`
}

// PersonGroupRequest 人员分组创建/更新请求。
type PersonGroupRequest struct {
	GroupName   string  `json:"group_name" binding:"required,min=1,max=128"`
	Description string  `json:"description" binding:"omitempty,max=255"`
	ParentID    *string `json:"parent_id"`
	SortOrder   int     `json:"sort_order"`
}

// PersonResponse 人员响应。
type PersonResponse struct {
	ID                       string                `json:"id"`
	PersonCode               string                `json:"person_code"`
	PersonName               string                `json:"person_name"`
	Gender                   string                `json:"gender"`
	Phone                    string                `json:"phone,omitempty"`
	IDNumber                 string                `json:"id_number,omitempty"`
	ImageURL                 string                `json:"image_url"`
	ImageMD5                 string                `json:"image_md5,omitempty"`
	FaceQualityScore         *float64              `json:"face_quality_score,omitempty"`
	EmbeddingStatus          string                `json:"embedding_status"`
	EmbeddingErrorCode       string                `json:"embedding_error_code,omitempty"`
	EmbeddingErrorMessageKey string                `json:"embedding_error_message_key,omitempty"`
	EmbeddingRetryable       bool                  `json:"embedding_retryable"`
	Enabled                  bool                  `json:"enabled"`
	Remark                   string                `json:"remark,omitempty"`
	Groups                   []PersonGroupResponse `json:"groups,omitempty"`
	CreatedAt                string                `json:"created_at"`
	UpdatedAt                string                `json:"updated_at"`
}

// PersonGroupResponse 人员分组响应。
type PersonGroupResponse struct {
	ID          string                `json:"id"`
	GroupName   string                `json:"group_name"`
	Description string                `json:"description,omitempty"`
	ParentID    string                `json:"parent_id,omitempty"`
	SortOrder   int                   `json:"sort_order"`
	PersonCount int64                 `json:"person_count"`
	Children    []PersonGroupResponse `json:"children,omitempty"`
	CreatedAt   string                `json:"created_at"`
	UpdatedAt   string                `json:"updated_at"`
}

// PersonActiveEmbeddingResponse 内部底库消费响应。
type PersonActiveEmbeddingResponse struct {
	PersonRecordID string                `json:"person_record_id"`
	PersonName     string                `json:"person_name"`
	Embedding      []float32             `json:"embedding"`
	Groups         []PersonGroupResponse `json:"groups,omitempty"`
}

// PersonImportByURLRequest 通过已上传文件路径导入人员（分片上传后调用）。
type PersonImportByURLRequest struct {
	FileURL              string `json:"file_url" binding:"required"`
	OverwriteOnDuplicate bool   `json:"overwrite_on_duplicate"`
}

// PersonImportTaskResponse 导入任务响应。
type PersonImportTaskResponse struct {
	ID            string `json:"id"`
	TaskType      string `json:"task_type"`
	FileName      string `json:"file_name"`
	FileURL       string `json:"file_url,omitempty"`
	TotalRows     int    `json:"total_rows"`
	SuccessRows   int    `json:"success_rows"`
	FailedRows    int    `json:"failed_rows"`
	FailDetailURL string `json:"fail_detail_url,omitempty"`
	Status        string `json:"status"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}
