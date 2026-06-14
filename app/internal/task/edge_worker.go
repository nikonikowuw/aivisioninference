package task

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/niko-admin/niko-admin/internal/repository"
	"github.com/niko-admin/niko-admin/internal/service"
)

// EdgeStateWorker reconciles edge node expected running tasks vs actual streams.
type EdgeStateWorker struct {
	taskRepo     *repository.AIVisionTaskRepository
	nodeRepo     *repository.EdgeNodeRepository
	rdb          *redis.Client
	engineClient service.EngineClient
}

// NewEdgeStateWorker creates a new EdgeStateWorker.
func NewEdgeStateWorker(
	taskRepo *repository.AIVisionTaskRepository,
	nodeRepo *repository.EdgeNodeRepository,
	rdb *redis.Client,
	engineClient service.EngineClient,
) *EdgeStateWorker {
	return &EdgeStateWorker{
		taskRepo:     taskRepo,
		nodeRepo:     nodeRepo,
		rdb:          rdb,
		engineClient: engineClient,
	}
}

// HandleReconcileEdgeState processes TaskReconcileEdgeState.
func (w *EdgeStateWorker) HandleReconcileEdgeState(ctx context.Context, t *asynq.Task) error {
	var payload struct {
		NodeID string `json:"node_id"`
	}
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		zap.L().Error("Asynq: failed to parse reconcile state payload", zap.Error(err))
		return err
	}

	nodeID := payload.NodeID
	zap.L().Info("Asynq: starting state reconciliation", zap.String("node_id", nodeID))

	// 1. Get expected tasks from DB
	runningTasks, err := w.taskRepo.FindRunningTasksByNode(ctx, nodeID)
	if err != nil {
		zap.L().Error("Asynq: failed to fetch running tasks for node", zap.String("node_id", nodeID), zap.Error(err))
		return err
	}

	expectedDeviceIDs := make(map[string]bool)
	for _, task := range runningTasks {
		expectedDeviceIDs[task.DeviceChannelID] = true
	}

	// 2. Get actual running streams from Redis
	streamsKey := fmt.Sprintf("aivision:edge:%s:actual_streams", nodeID)
	streamsBytes, err := w.rdb.Get(ctx, streamsKey).Bytes()
	if err == redis.Nil {
		zap.L().Info("Asynq: no actual streams found in Redis, nothing to reconcile", zap.String("node_id", nodeID))
		return nil
	} else if err != nil {
		zap.L().Error("Asynq: failed to fetch actual streams from Redis", zap.String("node_id", nodeID), zap.Error(err))
		return err
	}

	var actualDeviceIDs []string
	if err := json.Unmarshal(streamsBytes, &actualDeviceIDs); err != nil {
		zap.L().Error("Asynq: failed to deserialize actual streams", zap.String("node_id", nodeID), zap.Error(err))
		return err
	}

	// 3. Compare and stop ghost streams
	for _, devID := range actualDeviceIDs {
		if !expectedDeviceIDs[devID] {
			zap.L().Warn("Asynq: ghost stream detected, stopping", zap.String("node_id", nodeID), zap.String("device_id", devID))
			
			// Issue corrective StopStream MQTT command
			stopCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			if err := w.engineClient.StopStream(stopCtx, devID); err != nil {
				zap.L().Error("Asynq: failed to stop ghost stream", zap.String("device_id", devID), zap.Error(err))
			}
			cancel()
		}
	}

	return nil
}
