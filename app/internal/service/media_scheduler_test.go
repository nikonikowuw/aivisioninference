package service

import (
	"reflect"
	"testing"
	"time"
)

func TestSelectMediaNodeUsesPostAdmissionPreviewUtilization(t *testing.T) {
	now := time.Now()
	candidates := []MediaNodeCandidate{
		{
			NodeID: "node-busy", Online: true, Enabled: true, MetricsValid: true,
			MetricsAt: now, MetricsTTL: time.Minute,
			Capacity: 10, Usage: 8,
		},
		{
			NodeID: "node-balanced", Online: true, Enabled: true, MetricsValid: true,
			MetricsAt: now, MetricsTTL: time.Minute,
			Capacity: 10, Usage: 4,
		},
	}

	got := SelectMediaNode(now, candidates)
	if got.NodeID != "node-balanced" {
		t.Fatalf("selected node = %q, want node-balanced", got.NodeID)
	}
}

func TestSelectMediaNodeRejectsFullPreviewCapacity(t *testing.T) {
	now := time.Now()
	candidate := MediaNodeCandidate{
		NodeID: "node-a", Online: true, Enabled: true, MetricsValid: true,
		MetricsAt: now, MetricsTTL: time.Minute,
		Capacity: 2, Usage: 1, Pending: 1,
	}

	got := SelectMediaNode(now, []MediaNodeCandidate{candidate})
	want := []string{"preview_full"}
	if got.NodeID != "" || !reflect.DeepEqual(got.Rejections["node-a"], want) {
		t.Fatalf("decision = %#v, want rejection %v", got, want)
	}
}

func TestSelectMediaNodeRejectsUnknownAndStaleMetrics(t *testing.T) {
	now := time.Now()
	base := MediaNodeCandidate{
		Online: true, Enabled: true,
		MetricsAt: now, MetricsTTL: time.Minute,
		Capacity: 2,
	}
	unknown := base
	unknown.NodeID = "unknown"
	stale := base
	stale.NodeID = "stale"
	stale.MetricsValid = true
	stale.MetricsAt = now.Add(-2 * time.Minute)

	got := SelectMediaNode(now, []MediaNodeCandidate{unknown, stale})
	for _, nodeID := range []string{"unknown", "stale"} {
		if !reflect.DeepEqual(got.Rejections[nodeID], []string{"stale_metrics"}) {
			t.Fatalf("%s rejection = %v", nodeID, got.Rejections[nodeID])
		}
	}
}

func TestSelectMediaNodeUsesNodeIDAsStableTieBreaker(t *testing.T) {
	now := time.Now()
	makeCandidate := func(nodeID string) MediaNodeCandidate {
		return MediaNodeCandidate{
			NodeID: nodeID, Online: true, Enabled: true, MetricsValid: true,
			MetricsAt: now, MetricsTTL: time.Minute,
			Capacity: 2,
		}
	}

	got := SelectMediaNode(now, []MediaNodeCandidate{makeCandidate("node-b"), makeCandidate("node-a")})
	if got.NodeID != "node-a" {
		t.Fatalf("selected node = %q, want node-a", got.NodeID)
	}
}
