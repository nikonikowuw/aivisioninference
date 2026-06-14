package task

import (
	"context"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"

	"github.com/niko-admin/niko-admin/internal/model"
	"github.com/niko-admin/niko-admin/internal/repository"
)

// Task Type Constant
const (
	TypeEdgeNodeAlgorithmRetry = "edge_node:algorithm_retry"
)

// EdgeNodeAlgorithmRetryTask handles periodic checks and retries for failed algorithm deployments.
type EdgeNodeAlgorithmRetryTask struct {
	nodeAlgoRepo *repository.EdgeNodeAlgorithmRepository
	nodeRepo     *repository.EdgeNodeRepository
}

// NewEdgeNodeAlgorithmRetryTask creates a new EdgeNodeAlgorithmRetryTask.
func NewEdgeNodeAlgorithmRetryTask(
	nodeAlgoRepo *repository.EdgeNodeAlgorithmRepository,
	nodeRepo *repository.EdgeNodeRepository,
) *EdgeNodeAlgorithmRetryTask {
	return &EdgeNodeAlgorithmRetryTask{
		nodeAlgoRepo: nodeAlgoRepo,
		nodeRepo:     nodeRepo,
	}
}

// RegisterHandlers registers the retry task handler with Asynq.
func (h *EdgeNodeAlgorithmRetryTask) RegisterHandlers(mux *asynq.ServeMux) {
	mux.HandleFunc(TypeEdgeNodeAlgorithmRetry, h.handleAlgorithmRetry)
}

func (h *EdgeNodeAlgorithmRetryTask) handleAlgorithmRetry(ctx context.Context, t *asynq.Task) error {
	zap.L().Info("starting edge node algorithm deployment retry check")

	// Find failed deployments (status = failed, retry_count < 3)
	failedDeployments, err := h.nodeAlgoRepo.FindFailedWithRetries(ctx)
	if err != nil {
		zap.L().Error("failed to find failed deployments for retry", zap.Error(err))
		return err
	}

	for _, dep := range failedDeployments {
		// Verify node status
		node, err := h.nodeRepo.FindByID(ctx, dep.NodeID)
		if err != nil {
			zap.L().Error("failed to find node for failed deployment", zap.String("node_id", dep.NodeID), zap.Error(err))
			continue
		}

		if node.Status != model.NodeStatusOnline {
			// Node is offline, skip retry for now
			continue
		}

		// Calculate backoff delay
		delay := calculateRetryDelay(dep.RetryCount)
		nextRetry := dep.LastRetryAt
		if nextRetry == nil {
			nextRetry = &dep.UpdatedAt
		}

		if time.Now().Before(nextRetry.Add(delay)) {
			// Not time to retry yet
			continue
		}

		zap.L().Info("triggering automatic retry for algorithm deployment",
			zap.String("node_id", dep.NodeID),
			zap.String("algo_package_id", dep.AlgoPackageID),
			zap.Int("retry_count", dep.RetryCount+1),
		)

		now := time.Now()
		updates := map[string]interface{}{
			"status":        model.AlgoDeployPending,
			"retry_count":   dep.RetryCount + 1,
			"last_retry_at": &now,
			"error_message": fmt.Sprintf("自动重试第 %d 次", dep.RetryCount+1),
		}

		if err := h.nodeAlgoRepo.UpdateForRetry(ctx, dep.NodeID, dep.AlgoPackageID, updates); err != nil {
			zap.L().Error("failed to update deployment for retry", zap.String("node_id", dep.NodeID), zap.String("algo_package_id", dep.AlgoPackageID), zap.Error(err))
		}
	}

	return nil
}

func calculateRetryDelay(retryCount int) time.Duration {
	if retryCount < 0 {
		retryCount = 0
	}
	if retryCount > 2 {
		retryCount = 2
	}
	// Exponential backoff: 5m, 10m, 20m
	return 5 * time.Minute * time.Duration(1<<retryCount)
}
