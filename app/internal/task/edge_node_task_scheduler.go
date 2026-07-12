package task

import (
	"context"
	"encoding/json"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"

	"github.com/niko-admin/niko-admin/internal/service"
)

const (
	// TypeEdgeNodeTaskPoll is the task type for periodic edge node scheduled task evaluation.
	TypeEdgeNodeTaskPoll = "edge_node:task_poll"
)

// EdgeNodeTaskSchedulerHandler handles Asynq tasks for edge node task scheduling.
type EdgeNodeTaskSchedulerHandler struct {
	scheduledTaskSvc *service.EdgeNodeScheduledTaskService
	terminalSvc      *service.EdgeNodeTerminalService
}

// NewEdgeNodeTaskSchedulerHandler creates a new handler.
func NewEdgeNodeTaskSchedulerHandler(
	scheduledTaskSvc *service.EdgeNodeScheduledTaskService,
	terminalSvc *service.EdgeNodeTerminalService,
) *EdgeNodeTaskSchedulerHandler {
	return &EdgeNodeTaskSchedulerHandler{
		scheduledTaskSvc: scheduledTaskSvc,
		terminalSvc:      terminalSvc,
	}
}

// RegisterHandlers registers task handlers with the Asynq mux.
func (h *EdgeNodeTaskSchedulerHandler) RegisterHandlers(mux *asynq.ServeMux) {
	mux.HandleFunc(TypeEdgeNodeTaskPoll, h.handleTaskPoll)
	// Future: add individual task execution handlers if needed
}

// RegisterEdgeNodePeriodicTasks registers periodic tasks with the scheduler.
// Called at app startup to register the poll task to run every minute.
func RegisterEdgeNodePeriodicTasks(scheduler *asynq.Scheduler) error {
	if scheduler == nil {
		zap.L().Info("asynq scheduler not available, skipping edge node task poll registration")
		return nil
	}

	// Run task poll every minute to evaluate cron/one-shot tasks
	entryID, err := scheduler.Register("@every 1m", asynq.NewTask(TypeEdgeNodeTaskPoll, nil))
	if err != nil {
		return err
	}

	zap.L().Info("registered edge node task poll periodic job", zap.String("entry_id", entryID))
	return nil
}

// handleTaskPoll is called periodically to evaluate and trigger due scheduled tasks.
func (h *EdgeNodeTaskSchedulerHandler) handleTaskPoll(ctx context.Context, t *asynq.Task) error {
	zap.L().Debug("edge node task poll starting")

	// Trigger cron tasks across all nodes
	if h.scheduledTaskSvc != nil {
		h.scheduledTaskSvc.TriggerCronTasks(ctx)
	}

	zap.L().Debug("edge node task poll completed")
	return nil
}

// EnqueueTaskExecution enqueues a task execution request.
// This can be used for immediate execution requests from the API.
func EnqueueTaskExecution(client *asynq.Client, executionID, nodeID string) error {
	payload, err := json.Marshal(map[string]string{
		"execution_id": executionID,
		"node_id":      nodeID,
	})
	if err != nil {
		return err
	}

	info, err := client.Enqueue(asynq.NewTask("edge_node:exec_task", payload), asynq.MaxRetry(2))
	if err != nil {
		return err
	}

	zap.L().Info("enqueued task execution",
		zap.String("execution_id", executionID),
		zap.String("node_id", nodeID),
		zap.String("task_id", info.ID),
	)

	return nil
}
