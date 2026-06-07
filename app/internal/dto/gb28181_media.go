package dto

import "time"

// StartLiveRequest 启动 GB28181 实时预览请求
type StartLiveRequest struct {
	DeviceID string `json:"device_id" binding:"required"`
}

// StopLiveRequest 停止 GB28181 实时预览请求
type StopLiveRequest struct {
	DeviceID string `json:"device_id" binding:"required"`
	StreamID string `json:"stream_id" binding:"required"`
}

// PlayResponse 播放响应（实时预览/回放共用）
type PlayResponse struct {
	URL      string `json:"url"`
	Protocol string `json:"protocol"`
	StreamID string `json:"stream_id"`
}

// StartPlaybackRequest 启动 GB28181 录像回放请求
type StartPlaybackRequest struct {
	DeviceID  string    `json:"device_id" binding:"required"`
	StartTime time.Time `json:"start_time" binding:"required"`
	EndTime   time.Time `json:"end_time" binding:"required"`
}

// PlaybackControlRequest GB28181 回放控制请求
type PlaybackControlRequest struct {
	StreamID string  `json:"stream_id" binding:"required"`
	Action   string  `json:"action" binding:"required,oneof=pause resume scale seek"`
	Speed    float64 `json:"speed"`
	Stamp    int64   `json:"stamp"`
}

// StopPlaybackRequest 停止 GB28181 录像回放请求
type StopPlaybackRequest struct {
	DeviceID string `json:"device_id" binding:"required"`
	StreamID string `json:"stream_id" binding:"required"`
}
