package service

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/niko-admin/niko-admin/internal/dto"
)

func TestBuildEdgeNodeMetricsMapsHeartbeatFields(t *testing.T) {
	req := &dto.HeartbeatRequest{
		CPUUsage: 37.5, MemoryUsage: 25,
		CPULoad1m: 1.2, CPULoad5m: 0.8, CPULoad15m: 0.4,
		HardwareInfo: dto.HardwareInfo{TotalMemory: 8_000},
		Disks:        []dto.DiskInfo{{Path: "/", TotalBytes: 1_000, UsedBytes: 400, UsagePercent: 40}},
		NetRXBytes:   100, NetTXBytes: 200, NetRXSpeed: 10, NetTXSpeed: 20,
		Uptime: 60, ProcessCount: 12, ThreadCount: 34, Temperature: 45.5,
		CurrentLoad: 2, EngineVersion: "1.2.3", HALPlatform: "rkmpp",
	}

	metrics := buildEdgeNodeMetrics("node-1", req)

	require.Equal(t, int64(2_000), metrics.MemoryUsed)
	require.Equal(t, int64(8_000), metrics.MemoryTotal)
	require.Equal(t, int64(100), metrics.NetRXBytes)
	require.Equal(t, float64(20), metrics.NetTXSpeed)
	require.Equal(t, int32(12), metrics.ProcessCount)

	var disks []dto.DiskInfo
	require.NoError(t, json.Unmarshal(metrics.DiskUsage, &disks))
	require.Equal(t, req.Disks, disks)
}
