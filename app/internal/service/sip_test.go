package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGB28181DeviceIDRegex(t *testing.T) {
	tests := []struct {
		deviceID string
		valid    bool
	}{
		{"34020000001320000001", true},
		{"3402000000132000000", false},  // 19 digits
		{"340200000013200000012", false}, // 21 digits
		{"device123", false},
		{"", false},
	}

	for _, tt := range tests {
		matched := gb28181DeviceIDRegex.MatchString(tt.deviceID)
		assert.Equal(t, tt.valid, matched, "deviceID=%s", tt.deviceID)
	}
}

func TestBuildCatalogueResponse(t *testing.T) {
	tests := []struct {
		name     string
		deviceID string
		sn       string
		wantErr  bool
	}{
		{
			name:     "invalid format",
			deviceID: "invalid-device",
			sn:       "456",
			wantErr:  true,
		},
	}

	svc := &SIPService{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := svc.BuildCatalogueResponse(context.Background(), tt.deviceID, tt.sn)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Empty(t, result)
			} else {
				assert.NoError(t, err)
				assert.Contains(t, result, "<CmdType>Catalog</CmdType>")
			}
		})
	}
}

func TestSIPService_HandleRegister(t *testing.T) {
	svc := &SIPService{}

	// Invalid device ID should return error
	err := svc.HandleRegister(context.Background(), "invalid", "192.168.1.1", 5060)
	assert.Error(t, err)
}

func TestCheckHeartbeatTimeout(t *testing.T) {
	svc := &SIPService{}
	// nil repo should panic or return error
	assert.Panics(t, func() {
		svc.CheckHeartbeatTimeout(context.Background(), 180*time.Second)
	})
}
