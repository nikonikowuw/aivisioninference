package service

import (
	"testing"
	"time"

	"github.com/niko-admin/niko-admin/internal/pkg/controlproto"
)

func TestEngineMetricsStoreKeepsNodeSnapshotsIsolated(t *testing.T) {
	store := NewEngineMetricsStore(NewHistoryBuffer(10), nil, nil)
	store.UpdateNode("node-a", &controlproto.EngineMetricsSnapshot{PreviewInUse: 1})
	store.UpdateNode("node-b", &controlproto.EngineMetricsSnapshot{PreviewInUse: 2})

	nodeA, receivedAt := store.GetNodeLatest("node-a")
	if nodeA == nil || nodeA.PreviewInUse != 1 || receivedAt.IsZero() {
		t.Fatalf("node-a snapshot = %#v, received_at = %v", nodeA, receivedAt)
	}
	nodeB, _ := store.GetNodeLatest("node-b")
	if nodeB == nil || nodeB.PreviewInUse != 2 {
		t.Fatalf("node-b snapshot = %#v", nodeB)
	}

	nodeA.PreviewInUse = 99
	storedAgain, _ := store.GetNodeLatest("node-a")
	if storedAgain.PreviewInUse != 1 {
		t.Fatalf("stored snapshot mutated through caller: %#v", storedAgain)
	}
}

func TestEngineMetricsStoreRequiresFreshCollectionAndReceiveTimes(t *testing.T) {
	store := NewEngineMetricsStore(NewHistoryBuffer(10), nil, nil)
	now := time.Now()
	store.UpdateNode("node-a", &controlproto.EngineMetricsSnapshot{
		TimestampNS:          uint64(now.UnixNano()),
		PreviewCapacityValid: true,
		PreviewCapacity:      1,
	})
	checkAt := time.Now()

	if _, ok := store.GetFreshNodeMediaMetrics("node-a", checkAt, 15*time.Second); !ok {
		t.Fatal("fresh metrics rejected")
	}
	if _, ok := store.GetFreshNodeMediaMetrics("node-a", checkAt.Add(16*time.Second), 15*time.Second); ok {
		t.Fatal("stale metrics accepted")
	}

	store.UpdateNode("old-engine", &controlproto.EngineMetricsSnapshot{
		TimestampNS:          uint64(now.Add(-time.Minute).UnixNano()),
		PreviewCapacityValid: true,
		PreviewCapacity:      1,
	})
	if _, ok := store.GetFreshNodeMediaMetrics("old-engine", now, 15*time.Second); ok {
		t.Fatal("stale engine collection time accepted")
	}
}
