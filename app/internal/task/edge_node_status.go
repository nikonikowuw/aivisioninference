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
}

// NewEdgeNodeStatusTask creates a new EdgeNodeStatusTask.
func NewEdgeNodeStatusTask(
	nodeRepo *repository.EdgeNodeRepository,
	taskRepo *repository.AIVisionTaskRepository,
	hub *ws.Hub,
	timeoutSeconds int,
) *EdgeNodeStatusTask {
	return &EdgeNodeStatusTask{
		nodeRepo:       nodeRepo,
		taskRepo:       taskRepo,
		hub:            hub,
		timeoutSeconds: timeoutSeconds,
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
	scheduler.Register(fmt.Sprintf("@every %ds", intervalSec),
		asynq.NewTask(TypeEdgeNodeStatusCheck, nil))
	zap.L().Info("registered periodic edge node status check",
		zap.Int("interval_sec", intervalSec))
}

func (h *EdgeNodeStatusTask) handleEdgeNodeStatusCheck(ctx context.Context, t *asynq.Task) error {
	zap.L().Info("starting edge node status check", zap.Int("timeout_seconds", h.timeoutSeconds))

	cutoff := time.Now().Add(-time.Duration(h.timeoutSeconds) * time.Second)

	nodes, err := h.nodeRepo.FindTimedOutNodes(ctx, cutoff)
	if err != nil {
		zap.L().Error("failed to find timed out edge nodes", zap.Error(err))
		return err
	}

	if len(nodes) == 0 {
		return nil
	}

	var firstErr error
	for _, node := range nodes {
		zap.L().Warn("edge node heartbeat timed out, marking offline",
			zap.String("id", node.ID),
			zap.String("name", node.Name))

		if _, err := service.HandleNodeOffline(ctx, h.nodeRepo, h.taskRepo, h.hub, node,
			model.SuspendedReasonNodeOffline,
			"节点 %s 心跳超时，任务自动暂停", &cutoff, node.Name); err != nil {
			zap.L().Error("failed to process node offline",
				zap.String("node_id", node.ID),
				zap.Error(err),
			)
			if firstErr == nil {
				firstErr = err
			}
		}
	}

	return firstErr
}
