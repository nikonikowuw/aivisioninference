// Shared node-offline processing entry point used by both the periodic heartbeat
// timeout check (EdgeNodeStatusTask) and the MQTT LWT lifecycle handler.
//
// It marks the node as offline, suspends all running tasks with the given
// suspendedReason, and broadcasts WebSocket notifications.
package service

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
	"go.uber.org/zap"

	"github.com/niko-admin/niko-admin/internal/model"
	"github.com/niko-admin/niko-admin/internal/pkg/ws"
	"github.com/niko-admin/niko-admin/internal/repository"
)

// HandleNodeOffline marks a node as offline and suspends all running tasks with the specified reason.
// Both the heartbeat timeout check and MQTT LWT handler converge here for consistent offline processing.
// Returns whether this call performed the transition. Persistence is transactional and notifications
// are emitted only after the node and all affected tasks commit successfully.
func HandleNodeOffline(
	ctx context.Context,
	nodeRepo *repository.EdgeNodeRepository,
	taskRepo *repository.AIVisionTaskRepository,
	hub *ws.Hub,
	store HeartbeatStore,
	node model.EdgeNode,
	suspendedReason string,
	reasonFmt string,
	cutoff *time.Time,
	reasonArgs ...interface{},
) (bool, error) {
	reason := fmt.Sprintf(reasonFmt, reasonArgs...)

	var suspendedTasks []model.AIVisionTask
	transitioned := false
	err := nodeRepo.DB(ctx).Transaction(func(tx *gorm.DB) error {
		txNodeRepo := nodeRepo.WithTx(tx)
		txTaskRepo := taskRepo.WithTx(tx)

		var err error
		if cutoff == nil {
			transitioned, err = txNodeRepo.MarkOffline(ctx, node.ID)
		} else {
			transitioned, err = txNodeRepo.MarkOfflineIfTimedOut(ctx, node.ID, *cutoff)
		}
		if err != nil {
			return fmt.Errorf("failed to update edge node %s status to offline: %w", node.ID, err)
		}
		if !transitioned {
			return nil
		}

		suspendedTasks, err = txTaskRepo.FindRunningTasksByNode(ctx, node.ID)
		if err != nil {
			return fmt.Errorf("failed to find running tasks for offline node %s: %w", node.ID, err)
		}
		for _, task := range suspendedTasks {
			if err := txTaskRepo.SetSuspended(ctx, task.ID, suspendedReason, reason); err != nil {
				return fmt.Errorf("failed to suspend task %s for offline node %s: %w", task.ID, node.ID, err)
			}
		}
		return nil
	})
	if err != nil {
		return false, err
	}
	if !transitioned {
		return false, nil
	}

	// Remove from heartbeat store — node is now offline
	if err := store.Remove(ctx, node.ID); err != nil {
		zap.L().Warn("failed to remove node from heartbeat store",
			zap.String("node_id", node.ID), zap.Error(err))
	}

	if hub != nil {
		hub.Broadcast(&ws.Message{
			Type: ws.TopicEdgeNodeStatus,
			Payload: map[string]interface{}{
				"node_id": node.ID,
				"status":  model.NodeStatusOffline,
			},
		})
	}
	for _, task := range suspendedTasks {
		if hub != nil {
			hub.Broadcast(&ws.Message{
				Type: ws.TopicTaskStatus,
				Payload: map[string]interface{}{
					"task_id":          task.ID,
					"status":           model.TaskStatusSuspended,
					"suspended_reason": suspendedReason,
					"error_reason":     reason,
					"node_id":          node.ID,
				},
			})
		}
	}

	return true, nil
}
