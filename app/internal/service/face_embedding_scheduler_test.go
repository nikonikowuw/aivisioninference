package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/niko-admin/niko-admin/internal/model"
	"github.com/niko-admin/niko-admin/internal/repository"
)

type fakeEmbeddingLeaseStore struct {
	values map[string]string
}

func newFakeEmbeddingLeaseStore() *fakeEmbeddingLeaseStore {
	return &fakeEmbeddingLeaseStore{values: make(map[string]string)}
}

func (s *fakeEmbeddingLeaseStore) SetNX(_ context.Context, key string, value interface{}, _ time.Duration) *redis.BoolCmd {
	cmd := redis.NewBoolCmd(context.Background())
	if _, exists := s.values[key]; exists {
		cmd.SetVal(false)
		return cmd
	}
	s.values[key] = fmt.Sprint(value)
	cmd.SetVal(true)
	return cmd
}

func (s *fakeEmbeddingLeaseStore) Eval(_ context.Context, _ string, keys []string, args ...interface{}) *redis.Cmd {
	cmd := redis.NewCmd(context.Background())
	if len(keys) > 0 && len(args) > 0 && s.values[keys[0]] == fmt.Sprint(args[0]) {
		delete(s.values, keys[0])
		cmd.SetVal(int64(1))
		return cmd
	}
	cmd.SetVal(int64(0))
	return cmd
}

func (s *fakeEmbeddingLeaseStore) Get(_ context.Context, key string) *redis.StringCmd {
	cmd := redis.NewStringCmd(context.Background())
	if value, ok := s.values[key]; ok {
		cmd.SetVal(value)
		return cmd
	}
	cmd.SetErr(redis.Nil)
	return cmd
}

type warmupEngineStub struct {
	nodeAlgoRepo *repository.EdgeNodeAlgorithmRepository
	warmupCalls  []string
}

func (s *warmupEngineStub) StartStream(ctx context.Context, req StreamStartRequest) (StreamInfo, error) {
	return StreamInfo{}, nil
}

func (s *warmupEngineStub) StopStream(ctx context.Context, nodeID, deviceID string) error {
	return nil
}

func (s *warmupEngineStub) StartPlayback(ctx context.Context, req StreamStartRequest) (string, error) {
	return "", nil
}

func (s *warmupEngineStub) StopPlayback(ctx context.Context, nodeID, deviceID string) error {
	return nil
}

func (s *warmupEngineStub) GetStreamStatus(ctx context.Context, nodeID, deviceID string) (StreamStatus, error) {
	return StreamStatus{}, nil
}

func (s *warmupEngineStub) StartSelfCheck(ctx context.Context, downloadURL, token, algoName, version string) error {
	return nil
}

func (s *warmupEngineStub) WarmupAlgorithm(ctx context.Context, nodeID, algoName, algoVersion string) error {
	s.warmupCalls = append(s.warmupCalls, nodeID)
	return s.nodeAlgoRepo.UpdateForRetry(ctx, nodeID, "pkg-1", map[string]interface{}{
		"runtime_status": model.AlgoRuntimeReady,
	})
}

func (s *warmupEngineStub) UpdateFaceLibrary(ctx context.Context, nodeID, algoName string, faceLibraryJSON []byte) error {
	return nil
}

func (s *warmupEngineStub) ExtractFaceEmbedding(ctx context.Context, nodeID, algoName, algoVersion string, imageBytes []byte) (FaceEmbeddingResult, error) {
	return FaceEmbeddingResult{Success: true, Embedding: make([]float32, 512)}, nil
}

func TestFaceEmbeddingSchedulerEnsureReadyNodeWarmsInstalledNode(t *testing.T) {
	db := setupServiceTestDB(t)
	ctx := context.Background()

	nodeRepo := repository.NewEdgeNodeRepository(db)
	nodeAlgoRepo := repository.NewEdgeNodeAlgorithmRepository(db)
	require.NoError(t, nodeRepo.Create(ctx, &model.EdgeNode{
		BaseModel:         model.BaseModel{ID: "node-1"},
		Name:              "node-1",
		Endpoint:          "http://node-1",
		AuthToken:         "token-1",
		Status:            model.NodeStatusOnline,
		Enabled:           true,
		EmbeddingCapacity: 1,
	}))
	require.NoError(t, nodeAlgoRepo.Create(ctx, &model.EdgeNodeAlgorithm{
		BaseModel:         model.BaseModel{ID: "deploy-1"},
		NodeID:            "node-1",
		AlgoPackageID:     "pkg-1",
		Status:            model.AlgoDeployInstalled,
		RuntimeStatus:     model.AlgoRuntimeInstalled,
		SupportsEmbedding: true,
		EmbeddingCapacity: 1,
	}))

	scheduler := NewFaceEmbeddingSchedulerWithStore(nodeRepo, newFakeEmbeddingLeaseStore())
	engine := &warmupEngineStub{nodeAlgoRepo: nodeAlgoRepo}

	nodeID, err := scheduler.EnsureReadyNode(ctx, "face_recognition", "1.0.0", "pkg-1", engine)
	require.NoError(t, err)
	assert.Equal(t, "node-1", nodeID)
	assert.Equal(t, []string{"node-1"}, engine.warmupCalls)
}

func TestFaceEmbeddingSchedulerAcquireReleaseSlot(t *testing.T) {
	db := setupServiceTestDB(t)
	ctx := context.Background()

	nodeRepo := repository.NewEdgeNodeRepository(db)
	nodeAlgoRepo := repository.NewEdgeNodeAlgorithmRepository(db)
	require.NoError(t, nodeRepo.Create(ctx, &model.EdgeNode{
		BaseModel:         model.BaseModel{ID: "node-1"},
		Name:              "node-1",
		Endpoint:          "http://node-1",
		AuthToken:         "token-1",
		Status:            model.NodeStatusOnline,
		Enabled:           true,
		EmbeddingCapacity: 1,
	}))
	require.NoError(t, nodeAlgoRepo.Create(ctx, &model.EdgeNodeAlgorithm{
		BaseModel:         model.BaseModel{ID: "deploy-1"},
		NodeID:            "node-1",
		AlgoPackageID:     "pkg-1",
		Status:            model.AlgoDeployInstalled,
		RuntimeStatus:     model.AlgoRuntimeReady,
		SupportsEmbedding: true,
		EmbeddingCapacity: 1,
	}))

	scheduler := NewFaceEmbeddingSchedulerWithStore(nodeRepo, newFakeEmbeddingLeaseStore())
	reservation, err := scheduler.Acquire(ctx, "pkg-1")
	require.NoError(t, err)
	assert.Equal(t, "node-1", reservation.NodeID)

	_, err = scheduler.Acquire(ctx, "pkg-1")
	require.ErrorContains(t, err, "no embedding slots available")
	require.NoError(t, scheduler.Release(ctx, reservation))

	again, err := scheduler.Acquire(ctx, "pkg-1")
	require.NoError(t, err)
	assert.Equal(t, "node-1", again.NodeID)
}
