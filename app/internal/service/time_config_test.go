package service

import (
	"testing"
	"time"
)

func TestGetTimezone(t *testing.T) {
	// 获取当前时区
	tz := time.Now().Location().String()

	if tz == "" {
		t.Error("expected non-empty timezone")
	}
}

func TestGetRecommendedNTPServers(t *testing.T) {
	svc := NewTimeConfigService(nil)

	servers := svc.GetRecommendedNTPServers()

	if len(servers) == 0 {
		t.Error("expected non-empty recommended servers list")
	}

	// 验证包含常用服务器
	found := false
	for _, s := range servers {
		if s == "ntp.aliyun.com" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected ntp.aliyun.com in recommended servers")
	}
}
