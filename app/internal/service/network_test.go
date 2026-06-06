package service

import (
	"testing"
)

func TestNetworkService_GetInterfaces(t *testing.T) {
	// 这个测试需要真实的网络环境
	// 在 CI/CD 环境中可能需要跳过
	if testing.Short() {
		t.Skip("skipping network test in short mode")
	}
}

func TestIsVirtualInterface(t *testing.T) {
	tests := []struct {
		name     string
		iface    string
		expected bool
	}{
		{"loopback", "lo", true},
		{"docker bridge", "docker0", true},
		{"veth pair", "veth1234", true},
		{"bridge", "br-abc123", true},
		{"real interface", "eth0", false},
		{"wlan", "wlan0", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isVirtualInterface(tt.iface)
			if result != tt.expected {
				t.Errorf("isVirtualInterface(%q) = %v, want %v", tt.iface, result, tt.expected)
			}
		})
	}
}

func TestGenerateUUID(t *testing.T) {
	id1 := generateUUID()
	id2 := generateUUID()

	if id1 == "" {
		t.Error("expected non-empty UUID")
	}
	if id1 == id2 {
		t.Error("expected unique UUIDs")
	}
}
