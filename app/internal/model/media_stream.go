// Package model 定义系统数据模型，包含 GORM 结构体和数据库表映射。
package model

import (
	"time"
)

const (
	MediaStreamModePreview   = "preview"
	MediaStreamModeInference = "inference"
	MediaStreamModeMixed     = "mixed"
)

// MediaStream 表示 ZLM 媒体流映射，关联平台设备/任务与 ZLM 流标识和播放地址
type MediaStream struct {
	BaseModel
	DeviceID           string     `gorm:"type:uuid;not null;index" json:"device_id"`
	TaskID             *string    `gorm:"type:uuid;index" json:"task_id,omitempty"`
	NodeID             *string    `gorm:"type:uuid;index" json:"node_id,omitempty"`
	StreamMode         string     `gorm:"type:varchar(16);not null;default:preview;index;check:stream_mode IN ('preview','inference','mixed')" json:"stream_mode"`
	DecodeSlots        int        `gorm:"not null;default:0;check:decode_slots >= 0" json:"decode_slots"`
	EncodeSlots        int        `gorm:"not null;default:0;check:encode_slots >= 0" json:"encode_slots"`
	EgressBps          int64      `gorm:"not null;default:0;check:egress_bps >= 0" json:"egress_bps"`
	ZLMApp             string     `gorm:"type:varchar(64);not null;default:live;uniqueIndex:idx_media_streams_zlm" json:"zlm_app"`
	ZLMStream          string     `gorm:"type:varchar(128);not null;uniqueIndex:idx_media_streams_zlm" json:"zlm_stream"`
	ZLMVhost           string     `gorm:"type:varchar(128);default:__defaultVhost__;uniqueIndex:idx_media_streams_zlm" json:"zlm_vhost"`
	ZLMSchema          string     `gorm:"type:varchar(16);default:rtsp" json:"zlm_schema"`
	PlayURLRtsp        string     `gorm:"type:text" json:"play_url_rtsp,omitempty"`
	PlayURLRtmp        string     `gorm:"type:text" json:"play_url_rtmp,omitempty"`
	PlayURLFlv         string     `gorm:"type:text" json:"play_url_flv,omitempty"`
	PlayURLWsFlv       string     `gorm:"type:text" json:"play_url_ws_flv,omitempty"`
	PlayURLWebrtc      string     `gorm:"type:text" json:"play_url_webrtc,omitempty"`
	PlayURLHls         string     `gorm:"type:text" json:"play_url_hls,omitempty"`
	PlayURLMp4         string     `gorm:"type:text" json:"play_url_mp4,omitempty"`
	ExternalKey        string     `gorm:"type:varchar(200);uniqueIndex" json:"external_key,omitempty"`
	Status             string     `gorm:"type:varchar(16);default:inactive;check:status IN ('active','inactive','error')" json:"status"`
	ConsumerCount      int        `gorm:"default:0" json:"consumer_count"`
	LastConsumerReason string     `gorm:"type:varchar(64)" json:"last_consumer_reason"`
	StartedAt          *time.Time `json:"started_at,omitempty"`
	StoppedAt          *time.Time `json:"stopped_at,omitempty"`
}

// SortableFields 返回允许排序的字段列表
func (MediaStream) SortableFields() []string {
	return []string{"created_at", "status"}
}

// TableName 覆盖 MediaStream 的默认表名
func (MediaStream) TableName() string {
	return "media_streams"
}
