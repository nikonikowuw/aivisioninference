package model

import (
	"time"
)

// GB28181StreamSession tracks an active GB28181 INVITE session and its RTP relationship
type GB28181StreamSession struct {
	BaseModel
	StreamID     string     `gorm:"type:varchar(64);uniqueIndex;not null" json:"stream_id"`
	DeviceCode   string     `gorm:"type:varchar(64);index;not null" json:"device_code"`
	ChannelID    string     `gorm:"type:varchar(64);index;not null" json:"channel_id"`
	StreamType   string     `gorm:"type:varchar(16);default:live" json:"stream_type"` // live, playback
	ZLMRTPPort   int        `gorm:"not null" json:"zlm_rtp_port"`
	SIPCallID    string     `gorm:"type:varchar(128);index" json:"sip_call_id"`
	SSRC         string     `gorm:"type:varchar(32)" json:"ssrc"`
	Status       string     `gorm:"type:varchar(16);default:pending" json:"status"` // pending, active, stopped, failed
	RemoteIP     string     `gorm:"type:varchar(64)" json:"remote_ip"`
	RemotePort   int        `json:"remote_port"`
	StartTime    *time.Time `json:"start_time,omitempty"`
	EndTime      *time.Time `json:"end_time,omitempty"`
}

func (GB28181StreamSession) TableName() string {
	return "gb28181_stream_sessions"
}
