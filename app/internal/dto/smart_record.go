package dto

import "time"

// SmartRecordListRequest SmartRecord 列表请求
type SmartRecordListRequest struct {
	PageRequest
	RecordType string `form:"type"`
	DeviceID   string `form:"device_id"`
	AlarmType  string `form:"alarm_type"`
	AlarmLevel string `form:"alarm_level"`
	StartTime  string `form:"start_time"`
	EndTime    string `form:"end_time"`
}

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
