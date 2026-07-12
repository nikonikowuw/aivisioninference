package service

import (
	"reflect"
	"testing"
	"time"
)

func TestSelectMediaNodeUsesPostAdmissionMaximumUtilization(t *testing.T) {
	now := time.Now()
	candidates := []MediaNodeCandidate{
		{
			NodeID: "node-bandwidth-bound", Online: true, Enabled: true, MetricsValid: true,
			MetricsAt: now, MetricsTTL: time.Minute,
			Capacity: MediaResourceVector{DecodeSlots: 10, EncodeSlots: 10, EgressBPS: 100},
			Usage:    MediaResourceVector{DecodeSlots: 1, EncodeSlots: 1, EgressBPS: 90},
		},
		{
			NodeID: "node-balanced", Online: true, Enabled: true, MetricsValid: true,
			MetricsAt: now, MetricsTTL: time.Minute,
			Capacity: MediaResourceVector{DecodeSlots: 10, EncodeSlots: 10, EgressBPS: 100},
			Usage:    MediaResourceVector{DecodeSlots: 5, EncodeSlots: 2, EgressBPS: 20},
		},
	}

	got := SelectMediaNode(now, candidates, MediaResourceVector{DecodeSlots: 1, EncodeSlots: 1, EgressBPS: 5})
	if got.NodeID != "node-balanced" {
		t.Fatalf("selected node = %q, want node-balanced", got.NodeID)
	}
}

func TestSelectMediaNodeRejectsEveryExceededDimension(t *testing.T) {
	now := time.Now()
	candidate := MediaNodeCandidate{
		NodeID: "node-a", Online: true, Enabled: true, MetricsValid: true,
		MetricsAt: now, MetricsTTL: time.Minute,
		Capacity: MediaResourceVector{DecodeSlots: 1, EncodeSlots: 1, EgressBPS: 100},
		Usage:    MediaResourceVector{DecodeSlots: 1, EncodeSlots: 1, EgressBPS: 95},
	}

	got := SelectMediaNode(now, []MediaNodeCandidate{candidate}, MediaResourceVector{DecodeSlots: 1, EncodeSlots: 1, EgressBPS: 10})
	want := []string{"decode", "encode", "bandwidth"}
	if got.NodeID != "" || !reflect.DeepEqual(got.Rejections["node-a"], want) {
		t.Fatalf("decision = %#v, want rejection %v", got, want)
	}
}

func TestSelectMediaNodeRejectsUnknownAndStaleMetrics(t *testing.T) {
	now := time.Now()
	base := MediaNodeCandidate{
		Online: true, Enabled: true,
		MetricsAt: now, MetricsTTL: time.Minute,
		Capacity: MediaResourceVector{DecodeSlots: 2, EncodeSlots: 2, EgressBPS: 100},
	}
	unknown := base
	unknown.NodeID = "unknown"
	stale := base
	stale.NodeID = "stale"
	stale.MetricsValid = true
	stale.MetricsAt = now.Add(-2 * time.Minute)

	got := SelectMediaNode(now, []MediaNodeCandidate{unknown, stale}, MediaResourceVector{})
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
			Capacity: MediaResourceVector{DecodeSlots: 2, EncodeSlots: 2, EgressBPS: 100},
		}
	}

	got := SelectMediaNode(now, []MediaNodeCandidate{makeCandidate("node-b"), makeCandidate("node-a")}, MediaResourceVector{})
	if got.NodeID != "node-a" {
		t.Fatalf("selected node = %q, want node-a", got.NodeID)
	}
}
