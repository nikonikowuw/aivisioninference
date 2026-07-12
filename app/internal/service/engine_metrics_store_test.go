package service

import (
	"testing"
	"time"

	"github.com/niko-admin/niko-admin/internal/pkg/controlproto"
)

func TestEngineMetricsStoreKeepsNodeSnapshotsIsolated(t *testing.T) {
	store := NewEngineMetricsStore(NewHistoryBuffer(10))
	store.UpdateNode("node-a", &controlproto.EngineMetricsSnapshot{DecodeSlotsUsed: 1})
	store.UpdateNode("node-b", &controlproto.EngineMetricsSnapshot{DecodeSlotsUsed: 2})

	nodeA, receivedAt := store.GetNodeLatest("node-a")
	if nodeA == nil || nodeA.DecodeSlotsUsed != 1 || receivedAt.IsZero() {
		t.Fatalf("node-a snapshot = %#v, received_at = %v", nodeA, receivedAt)
	}
	nodeB, _ := store.GetNodeLatest("node-b")
	if nodeB == nil || nodeB.DecodeSlotsUsed != 2 {
		t.Fatalf("node-b snapshot = %#v", nodeB)
	}

	nodeA.DecodeSlotsUsed = 99
	storedAgain, _ := store.GetNodeLatest("node-a")
	if storedAgain.DecodeSlotsUsed != 1 {
		t.Fatalf("stored snapshot mutated through caller: %#v", storedAgain)
	}
}

func TestEngineMetricsStoreRequiresFreshCollectionAndReceiveTimes(t *testing.T) {
	store := NewEngineMetricsStore(NewHistoryBuffer(10))
	now := time.Now()
	store.UpdateNode("node-a", &controlproto.EngineMetricsSnapshot{
		TimestampNS:       uint64(now.UnixNano()),
		MediaMetricsValid: true,
	})
	checkAt := time.Now()

	if _, ok := store.GetFreshNodeMediaMetrics("node-a", checkAt, 15*time.Second); !ok {
		t.Fatal("fresh metrics rejected")
	}
	if _, ok := store.GetFreshNodeMediaMetrics("node-a", checkAt.Add(16*time.Second), 15*time.Second); ok {
		t.Fatal("stale metrics accepted")
	}

	store.UpdateNode("old-engine", &controlproto.EngineMetricsSnapshot{
		TimestampNS:       uint64(now.Add(-time.Minute).UnixNano()),
		MediaMetricsValid: true,
	})
	if _, ok := store.GetFreshNodeMediaMetrics("old-engine", now, 15*time.Second); ok {
		t.Fatal("stale engine collection time accepted")
	}
}
