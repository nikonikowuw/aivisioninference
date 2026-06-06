package service

import (
	"testing"
)

func TestContainsTag(t *testing.T) {
	tests := []struct {
		tags     []string
		tag      string
		expected bool
	}{
		{[]string{"alarm", "recognition"}, "alarm", true},
		{[]string{"alarm", "recognition"}, "capture", false},
		{[]string{"Alarm"}, "alarm", true}, // 大小写不敏感
		{[]string{}, "alarm", false},
	}

	for _, tt := range tests {
		result := containsTag(tt.tags, tt.tag)
		if result != tt.expected {
			t.Errorf("containsTag(%v, %q): expected %v, got %v", tt.tags, tt.tag, tt.expected, result)
		}
	}
}

func TestWebhookPushRequest(t *testing.T) {
	req := WebhookPushRequest{
		DeviceSn:            "DEVICE-001",
		CameraCode:          "CAM-001",
		CaptureTime:         "2026-06-05T10:30:00+08:00",
		SnapshotImagePath:   "/snapshots/2026/06/05/xxx.jpg",
		BackgroundImagePath: "/backgrounds/2026/06/05/yyy.jpg",
		AlarmMajor:          "alarm",
		AlarmType:           10004,
		Confidence:          0.95,
	}

	if req.DeviceSn != "DEVICE-001" {
		t.Errorf("expected DeviceSn to be DEVICE-001, got %s", req.DeviceSn)
	}

	if req.AlarmType != 10004 {
		t.Errorf("expected AlarmType to be 10004, got %d", req.AlarmType)
	}
}
