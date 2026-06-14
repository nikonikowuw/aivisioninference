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
	
	var ids []string
	for _, node := range nodes {
		zap.L().Warn("edge node heartbeat timed out, marking offline", 
			zap.String("id", node.ID), 
			zap.String("name", node.Name))
		ids = append(ids, node.ID)

		h.broadcastNodeOffline(node.ID)
		h.suspendNodeTasks(ctx, node)
	}
	
	if err := h.nodeRepo.UpdateStatusBatch(ctx, ids, model.NodeStatusOffline); err != nil {
		zap.L().Error("failed to update edge node status batch", zap.Error(err))
		return err
	}
	
	return nil
}

func (h *EdgeNodeStatusTask) broadcastNodeOffline(nodeID string) {
	if h.hub == nil {
		return
	}
	h.hub.Broadcast(&ws.Message{
		Type: "edge-node-status",
		Payload: map[string]interface{}{
			"node_id": nodeID,
			"status":  model.NodeStatusOffline,
		},
	})
}

func (h *EdgeNodeStatusTask) suspendNodeTasks(ctx context.Context, node model.EdgeNode) {
	runningTasks, err := h.taskRepo.FindRunningTasksByNode(ctx, node.ID)
	if err != nil {
		zap.L().Error("failed to find running tasks for offline node",
			zap.String("node_id", node.ID),
			zap.Error(err),
		)
		return
	}

	for _, task := range runningTasks {
		reason := fmt.Sprintf("节点 %s 离线，任务自动暂停", node.Name)
		if err := h.taskRepo.SetErrorReason(ctx, task.ID, reason); err != nil {
			zap.L().Error("failed to suspend task for offline node",
				zap.String("task_id", task.ID),
				zap.String("node_id", node.ID),
				zap.Error(err),
			)
			continue
		}

		zap.L().Info("task suspended due to node offline",
			zap.String("task_id", task.ID),
			zap.String("node_id", node.ID),
		)

		if h.hub != nil {
			h.hub.Broadcast(&ws.Message{
				Type: "task-status",
				Payload: map[string]interface{}{
					"task_id":      task.ID,
					"status":       model.TaskStatusSuspended,
					"error_reason": reason,
					"node_id":      node.ID,
				},
			})
		}
	}
}
