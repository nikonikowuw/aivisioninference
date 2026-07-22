package service

import (
	"testing"
)

func TestDeriveSubStreamURL(t *testing.T) {
	tests := []struct {
		name     string
		mainURL  string
		expected string
	}{
		{
			name:     "Hikvision 101 to 102",
			mainURL:  "rtsp://admin:pass@192.168.1.64:554/Streaming/Channels/101",
			expected: "rtsp://admin:pass@192.168.1.64:554/Streaming/Channels/102",
		},
		{
			name:     "Dahua subtype=0 to subtype=1",
			mainURL:  "rtsp://admin:pass@192.168.1.64:554/cam/realmonitor?channel=1&subtype=0",
			expected: "rtsp://admin:pass@192.168.1.64:554/cam/realmonitor?channel=1&subtype=1",
		},
		{
			name:     "Uniview s0 to s1",
			mainURL:  "rtsp://admin:pass@192.168.1.64:554/unicast/c1/s0/live",
			expected: "rtsp://admin:pass@192.168.1.64:554/unicast/c1/s1/live",
		},
		{
			name:     "TP-Link stream1 to stream2",
			mainURL:  "rtsp://admin:pass@192.168.1.64:554/stream1",
			expected: "rtsp://admin:pass@192.168.1.64:554/stream2",
		},
		{
			name:     "Generic main to sub",
			mainURL:  "rtsp://admin:pass@192.168.1.64:554/live/main/ch0",
			expected: "rtsp://admin:pass@192.168.1.64:554/live/sub/ch0",
		},
		{
			name:     "Unknown format fallback to main",
			mainURL:  "rtsp://admin:pass@192.168.1.64:554/custom_stream",
			expected: "rtsp://admin:pass@192.168.1.64:554/custom_stream",
		},
		{
			name:     "Empty URL",
			mainURL:  "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DeriveSubStreamURL(tt.mainURL)
			if got != tt.expected {
				t.Errorf("DeriveSubStreamURL(%q) = %q, want %q", tt.mainURL, got, tt.expected)
			}
		})
	}
}
