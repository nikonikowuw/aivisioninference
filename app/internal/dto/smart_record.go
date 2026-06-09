package dto

import (
	"time"

	"github.com/niko-admin/niko-admin/internal/pkg/scopes"
)

// SmartRecordListRequest 是智能记录列表查询请求。
type SmartRecordListRequest struct {
	PageRequest
	Keyword       string     `form:"keyword"`
	RecordType    string     `form:"type"`
	DeviceID      string     `form:"device_id"`
	GroupID       string     `form:"group_id"`
	TaskID        string     `form:"task_id"`
	DeviceName    string     `form:"device_name"`
	TaskName      string     `form:"task_name"`
	AlarmType     string     `form:"alarm_type"`
	AlarmLevel    string     `form:"alarm_level"`
	AlarmStatus   string     `form:"alarm_status"`
	PersonName    string     `form:"person_name"`
	BusinessTag   string     `form:"business_tag"`
	CategoryCode  *int       `form:"category_code"`
	MinConfidence *float64   `form:"min_confidence"`
	MaxConfidence *float64   `form:"max_confidence"`
	MinSimilarity *float64   `form:"min_similarity"`
	MaxSimilarity *float64   `form:"max_similarity"`
	StartTime     string     `form:"start_time"`
	EndTime       string     `form:"end_time"`
	FromTime      *time.Time `form:"-" json:"-"`
	ToTime        *time.Time `form:"-" json:"-"`
}

// UpdateSmartRecordAlarmStatusRequest 是告警记录处理状态更新请求。
type UpdateSmartRecordAlarmStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=unhandled handled"`
}

// CategoryCodeOption 表示类别下拉选项，供前端下拉选择使用。
type CategoryCodeOption struct {
	Value int    `json:"value"`
	Label string `json:"label"`
}

// SmartRecordResponse 是智能记录响应 DTO。
type SmartRecordResponse struct {
	RecordID         string      `json:"record_id"`
	RecordType       string      `json:"record_type"`
	DeviceID         string      `json:"device_id,omitempty"`
	DeviceName       string      `json:"device_name,omitempty"`
	TaskName         string      `json:"task_name,omitempty"`
	AlarmType        string      `json:"alarm_type,omitempty"`
	AlarmLevel       string      `json:"alarm_level,omitempty"`
	SnapshotImageURL string      `json:"snapshot_image_url,omitempty"`
	Confidence       *float64    `json:"confidence,omitempty"`
	RawResult        interface{} `json:"raw_result,omitempty"`
	CreatedAt        time.Time   `json:"created_at"`
}

// FilterScopes 返回智能记录列表过滤条件。
func (r *SmartRecordListRequest) FilterScopes() []scopes.Scope {
	var sc []scopes.Scope
	if r.Keyword != "" {
		sc = append(sc, scopes.MultiLike([]string{"device_name", "task_name", "algorithm_name", "person_name", "alarm_type", "alarm_major"}, r.Keyword))
	}
	if r.RecordType != "" {
		sc = append(sc, scopes.Eq("record_type", r.RecordType))
	}
	if r.DeviceID != "" {
		sc = append(sc, scopes.Eq("device_id", r.DeviceID))
	}
	if r.TaskID != "" {
		sc = append(sc, scopes.Eq("task_id", r.TaskID))
	}
	if r.DeviceName != "" {
		sc = append(sc, scopes.Like("device_name", r.DeviceName))
	}
	if r.TaskName != "" {
		sc = append(sc, scopes.Like("task_name", r.TaskName))
	}
	if r.AlarmType != "" {
		sc = append(sc, scopes.Eq("alarm_type", r.AlarmType))
	}
	if r.AlarmLevel != "" {
		sc = append(sc, scopes.Eq("alarm_level", r.AlarmLevel))
	}
	if r.AlarmStatus != "" {
		sc = append(sc, scopes.Eq("alarm_status", r.AlarmStatus))
	}
	if r.PersonName != "" {
		sc = append(sc, scopes.Like("person_name", r.PersonName))
	}
	if r.CategoryCode != nil {
		sc = append(sc, scopes.Eq("category_code", *r.CategoryCode))
	}
	if r.FromTime != nil || r.ToTime != nil {
		sc = append(sc, scopes.TimeRange("capture_time", r.FromTime, r.ToTime))
	}
	return sc
}
