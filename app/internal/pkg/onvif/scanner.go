package onvif

import (
	"context"
	"time"
)

type DeviceInfo struct {
	IP           string
	MAC          string
	Manufacturer string
	Model        string
	StreamURL    string
}

type Scanner struct{}

func NewScanner() *Scanner {
	return &Scanner{}
}

// Scan 发送 WS-Discovery 探测包并收集响应
func (s *Scanner) Scan(ctx context.Context, netInterface string, timeout time.Duration) ([]DeviceInfo, error) {
	// TODO: 真正的 ONVIF 扫描逻辑 (github.com/use-go/onvif)
	// 这里先返回模拟数据
	return []DeviceInfo{
		{
			IP:           "192.168.1.100",
			MAC:          "00:11:22:33:44:55",
			Manufacturer: "Hikvision",
			Model:        "DS-2CD2T47G2",
			StreamURL:    "rtsp://192.168.1.100:554/Streaming/Channels/101",
		},
	}, nil
}
