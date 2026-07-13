package task

import (
	"context"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"

	"github.com/niko-admin/niko-admin/internal/model"
	"github.com/niko-admin/niko-admin/internal/pkg/ws"
	"github.com/niko-admin/niko-admin/internal/repository"
	"github.com/niko-admin/niko-admin/internal/service"
)

// Task Type Constant
const (
	TypeEdgeNodeStatusCheck = "edge_node:status_check"
)

// EdgeNodeStatusTask handles periodic checks for edge nodes statuses.
type EdgeNodeStatusTask struct {
	nodeRepo       *repository.EdgeNodeRepository
	taskRepo       *repository.AIVisionTaskRepository
	hub            *ws.Hub
	timeoutSeconds int
	store          service.HeartbeatStore
}

// NewEdgeNodeStatusTask creates a new EdgeNodeStatusTask.
func NewEdgeNodeStatusTask(
	nodeRepo *repository.EdgeNodeRepository,
	taskRepo *repository.AIVisionTaskRepository,
	hub *ws.Hub,
	timeoutSeconds int,
	store service.HeartbeatStore,
) *EdgeNodeStatusTask {
	return &EdgeNodeStatusTask{
		nodeRepo:       nodeRepo,
		taskRepo:       taskRepo,
		hub:            hub,
		timeoutSeconds: timeoutSeconds,
		store:          store,
	}
}

// RegisterHandlers registers the status check handler with Asynq.
func (h *EdgeNodeStatusTask) RegisterHandlers(mux *asynq.ServeMux) {
	mux.HandleFunc(TypeEdgeNodeStatusCheck, h.handleEdgeNodeStatusCheck)
}

// RegisterPeriodic registers the periodic schedule for the status check task.
func (h *EdgeNodeStatusTask) RegisterPeriodic(scheduler *asynq.Scheduler, intervalSec int) {
	if intervalSec <= 0 {
		zap.L().Warn("edge node status check interval not positive, skipping periodic registration",
			zap.Int("interval_sec", intervalSec))
		return
	}
	if _, err := scheduler.Register(fmt.Sprintf("@every %ds", intervalSec),
		asynq.NewTask(TypeEdgeNodeStatusCheck, nil)); err != nil {
		zap.L().Error("failed to register periodic edge node status check",
			zap.Int("interval_sec", intervalSec),
			zap.Error(err))
		return
	}
	zap.L().Info("registered periodic edge node status check",
		zap.Int("interval_sec", intervalSec))
}

func (h *EdgeNodeStatusTask) handleEdgeNodeStatusCheck(ctx context.Context, t *asynq.Task) error {
	zap.L().Info("starting edge node status check", zap.Int("timeout_seconds", h.timeoutSeconds))

	cutoff := time.Now().Add(-time.Duration(h.timeoutSeconds) * time.Second)

	// Query HeartbeatStore for expired nodes (Redis ZSET or in-memory).
	expiredIDs, err := h.store.GetExpired(ctx, cutoff)
	if err != nil {
		zap.L().Error("failed to query heartbeat store for expired nodes", zap.Error(err))
		return err
	}

	if len(expiredIDs) == 0 {
		return nil
	}

	// Batch load nodes from DB to verify they still exist and are online.
	nodes, err := h.nodeRepo.FindByIDs(ctx, expiredIDs)
	if err != nil {
		zap.L().Error("failed to load nodes by IDs", zap.Error(err))
		return err
	}

	nodeMap := make(map[string]model.EdgeNode, len(nodes))
	for _, n := range nodes {
		nodeMap[n.ID] = n
	}

	var firstErr error
	for _, nodeID := range expiredIDs {
		node, ok := nodeMap[nodeID]
		if !ok || node.Status != model.NodeStatusOnline {
			// Node doesn't exist or is already offline — clean up stale store entry.
			if err := h.store.Remove(ctx, nodeID); err != nil {
				zap.L().Warn("failed to remove stale heartbeat entry from store",
					zap.String("node_id", nodeID), zap.Error(err))
			}
			continue
		}

		zap.L().Warn("edge node heartbeat timed out, marking offline",
			zap.String("id", node.ID),
			zap.String("name", node.Name))

		if _, err := service.HandleNodeOffline(ctx, h.nodeRepo, h.taskRepo, h.hub, h.store, node,
			model.SuspendedReasonNodeOffline,
			"节点 %s 心跳超时，任务自动暂停", &cutoff, node.Name); err != nil {
			zap.L().Error("failed to process node offline",
				zap.String("node_id", node.ID),
				zap.Error(err),
			)
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		// HandleNodeOffline removes from store on successful transition.
		// For non-transitioned (concurrent heartbeat refreshed DB), the store
		// entry was also refreshed by the concurrent Record call — leave it.
	}

	return firstErr
}
