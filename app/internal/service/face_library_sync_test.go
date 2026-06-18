package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type blockingFaceLibraryEngine struct {
	MockEngineClient

	started chan string
	release <-chan struct{}
}

func (e *blockingFaceLibraryEngine) UpdateFaceLibrary(ctx context.Context, nodeID, algoName string, faceLibraryJSON []byte) error {
	e.started <- nodeID
	select {
	case <-e.release:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

type failingFaceLibraryEngine struct {
	MockEngineClient

	failed map[string]error
}

func (e *failingFaceLibraryEngine) UpdateFaceLibrary(ctx context.Context, nodeID, algoName string, faceLibraryJSON []byte) error {
	return e.failed[nodeID]
}

func TestFaceLibrarySyncUpdatesNodesConcurrentlyWithLimit(t *testing.T) {
	release := make(chan struct{})
	engine := &blockingFaceLibraryEngine{
		started: make(chan string, maxFaceLibrarySyncConcurrency+1),
		release: release,
	}
	svc := &FaceLibrarySyncService{engine: engine}
	nodeIDs := make([]string, 0, maxFaceLibrarySyncConcurrency+4)
	for i := 0; i < maxFaceLibrarySyncConcurrency+4; i++ {
		nodeIDs = append(nodeIDs, fmt.Sprintf("node-%02d", i))
	}

	done := make(chan []string, 1)
	go func() {
		done <- svc.updateFaceLibraryOnNodes(context.Background(), nodeIDs, "face_recognition", []byte("{}"))
	}()

	for i := 0; i < maxFaceLibrarySyncConcurrency; i++ {
		select {
		case <-engine.started:
		case <-time.After(time.Second):
			t.Fatalf("expected worker %d to start", i+1)
		}
	}

	select {
	case nodeID := <-engine.started:
		t.Fatalf("started more than %d concurrent updates, extra node %s", maxFaceLibrarySyncConcurrency, nodeID)
	case <-time.After(30 * time.Millisecond):
	}

	close(release)
	select {
	case failed := <-done:
		assert.Empty(t, failed)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for face library updates")
	}
}

func TestFaceLibrarySyncReturnsFailuresInNodeOrder(t *testing.T) {
	engine := &failingFaceLibraryEngine{failed: map[string]error{
		"node-2": fmt.Errorf("boom 2"),
		"node-4": fmt.Errorf("boom 4"),
	}}
	svc := &FaceLibrarySyncService{engine: engine}

	failed := svc.updateFaceLibraryOnNodes(context.Background(), []string{
		"node-1",
		"node-2",
		"node-3",
		"node-4",
	}, "face_recognition", []byte("{}"))

	require.Equal(t, []string{
		"node-2:boom 2",
		"node-4:boom 4",
	}, failed)
}
