package service

import (
	"testing"
)

func TestNPUCollector_Mock(t *testing.T) {
	collector := NewMockCollector()

	metrics, err := collector.Collect()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if metrics.Supported {
		t.Error("expected Supported to be false for mock collector")
	}

	if metrics.Usage != 0 {
		t.Errorf("expected Usage to be 0, got %f", metrics.Usage)
	}

	if metrics.Temperature != 0 {
		t.Errorf("expected Temperature to be 0, got %f", metrics.Temperature)
	}
}

func TestDetectHardware(t *testing.T) {
	hardware := detectHardware()

	// 在测试环境中，应该返回 unknown 或实际硬件
	if hardware == "" {
		t.Error("expected non-empty hardware string")
	}
}

func TestParseNTPServers(t *testing.T) {
	servers := []string{"ntp.aliyun.com", "cn.ntp.org.cn"}

	result := parseNTPServers(servers)

	if len(result) != 2 {
		t.Errorf("expected 2 servers, got %d", len(result))
	}

	if result[0].Host != "ntp.aliyun.com" {
		t.Errorf("expected first server to be ntp.aliyun.com, got %s", result[0].Host)
	}

	if result[0].Status != "unknown" {
		t.Errorf("expected first server status to be unknown, got %s", result[0].Status)
	}
}
