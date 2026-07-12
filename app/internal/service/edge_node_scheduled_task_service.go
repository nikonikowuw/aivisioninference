package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/model"
	"github.com/niko-admin/niko-admin/internal/pkg/mqttsync"
	"github.com/niko-admin/niko-admin/internal/repository"
)

// EdgeNodeScheduledTaskService handles business logic for scheduled tasks.
type EdgeNodeScheduledTaskService struct {
	taskRepo *repository.EdgeNodeScheduledTaskRepository
	execRepo *repository.EdgeNodeTaskExecutionRepository
	nodeRepo *repository.EdgeNodeRepository
	mqttClient mqtt.Client
	syncManager *mqttsync.MqttSyncManager
	mu sync.Mutex
}

// NewEdgeNodeScheduledTaskService creates a new EdgeNodeScheduledTaskService.
func NewEdgeNodeScheduledTaskService(
	taskRepo *repository.EdgeNodeScheduledTaskRepository,
	execRepo *repository.EdgeNodeTaskExecutionRepository,
	nodeRepo *repository.EdgeNodeRepository,
	mqttClient mqtt.Client,
	syncManager *mqttsync.MqttSyncManager,
) *EdgeNodeScheduledTaskService {
	return &EdgeNodeScheduledTaskService{
		taskRepo:   taskRepo,
		execRepo:   execRepo,
		nodeRepo:   nodeRepo,
		mqttClient: mqttClient,
		syncManager: syncManager,
	}
}

// Create creates a new scheduled task.
func (s *EdgeNodeScheduledTaskService) Create(ctx context.Context, nodeID string, req dto.ScheduledTaskRequest) (*model.EdgeNodeScheduledTask, error) {
	// Validate node exists
	if _, err := s.nodeRepo.FindByID(ctx, nodeID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("节点不存在")
		}
		return nil, fmt.Errorf("查询节点失败: %w", err)
	}

	timeout := req.TimeoutSeconds
	if timeout <= 0 {
		timeout = 30
	}

	task := &model.EdgeNodeScheduledTask{
		BaseModel:      model.BaseModel{ID: uuid.New().String()},
		NodeID:         nodeID,
		Name:           req.Name,
		Command:        req.Command,
		CronExpr:       req.CronExpr,
		Status:         model.ScheduledTaskActive,
		TimeoutSeconds: timeout,
	}

	if err := s.taskRepo.Create(ctx, task); err != nil {
		return nil, fmt.Errorf("创建定时任务失败: %w", err)
	}

	return task, nil
}

// List returns paginated scheduled tasks for a node.
func (s *EdgeNodeScheduledTaskService) List(ctx context.Context, nodeID string, page, pageSize int, status string) ([]model.EdgeNodeScheduledTask, int64, error) {
	return s.taskRepo.ListByNode(ctx, nodeID, page, pageSize, status)
}

// GetByID returns a scheduled task by ID.
func (s *EdgeNodeScheduledTaskService) GetByID(ctx context.Context, id string) (*model.EdgeNodeScheduledTask, error) {
	task, err := s.taskRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("任务不存在")
		}
		return nil, fmt.Errorf("查询任务失败: %w", err)
	}
	return task, nil
}

// Update updates a scheduled task.
func (s *EdgeNodeScheduledTaskService) Update(ctx context.Context, id string, req dto.ScheduledTaskUpdateRequest) (*model.EdgeNodeScheduledTask, error) {
	task, err := s.taskRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("任务不存在")
		}
		return nil, fmt.Errorf("查询任务失败: %w", err)
	}

	if req.Name != nil {
		task.Name = *req.Name
	}
	if req.Command != nil {
		task.Command = *req.Command
	}
	if req.CronExpr != nil {
		task.CronExpr = *req.CronExpr
	}
	if req.TimeoutSeconds != nil {
		task.TimeoutSeconds = *req.TimeoutSeconds
	}
	if req.Status != nil {
		task.Status = *req.Status
	}

	if err := s.taskRepo.Update(ctx, task); err != nil {
		return nil, fmt.Errorf("更新任务失败: %w", err)
	}

	return task, nil
}

// Delete deletes a scheduled task.
func (s *EdgeNodeScheduledTaskService) Delete(ctx context.Context, id string) error {
	if _, err := s.taskRepo.FindByID(ctx, id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("任务不存在")
		}
		return fmt.Errorf("查询任务失败: %w", err)
	}
	return s.taskRepo.Delete(ctx, id)
}

// ListExecutions returns paginated execution records for a task.
func (s *EdgeNodeScheduledTaskService) ListExecutions(ctx context.Context, taskID string, page, pageSize int, status string) ([]model.EdgeNodeTaskExecution, int64, error) {
	return s.execRepo.ListByTask(ctx, taskID, page, pageSize, status)
}

// ExecuteNow sends a shell_exec command to the engine immediately.
// This is used for one-shot manual execution or triggered by the cron scheduler.
func (s *EdgeNodeScheduledTaskService) ExecuteNow(ctx context.Context, task *model.EdgeNodeScheduledTask) (*model.EdgeNodeTaskExecution, error) {
	// Create execution record
	now := time.Now()
	exec := &model.EdgeNodeTaskExecution{
		BaseModel: model.BaseModel{ID: uuid.New().String()},
		TaskID:    task.ID,
		Status:    model.TaskExecRunning,
		StartedAt: &now,
	}

	if err := s.execRepo.Create(ctx, exec); err != nil {
		return nil, fmt.Errorf("创建执行记录失败: %w", err)
	}

	// Send MQTT shell_exec command
	traceID := uuid.New().String()
	topic := fmt.Sprintf("aivision/edge/%s/cmd/shell_exec", task.NodeID)

	payload := map[string]interface{}{
		"trace_id":        traceID,
		"execution_id":    exec.ID,
		"command":         task.Command,
		"timeout_seconds": task.TimeoutSeconds,
		"callback_url":    fmt.Sprintf("/api/v1/edge-nodes/%s/task-executions/callback", task.NodeID),
	}
	payloadBytes, _ := json.Marshal(payload)

	if s.mqttClient == nil {
		// Update execution as failed if no MQTT
		finished := time.Now()
		_ = s.execRepo.UpdateResult(ctx, exec.ID, model.TaskExecFailed, "", "MQTT client not available", -1, finished)
		return exec, fmt.Errorf("MQTT client not available")
	}

	tokenPub := s.mqttClient.Publish(topic, 1, false, payloadBytes)
	tokenPub.Wait()
	if tokenPub.Error() != nil {
		finished := time.Now()
		_ = s.execRepo.UpdateResult(ctx, exec.ID, model.TaskExecFailed, "", tokenPub.Error().Error(), -1, finished)
		return exec, fmt.Errorf("MQTT publish failed: %w", tokenPub.Error())
	}

	zap.L().Info("shell_exec command sent to engine",
		zap.String("node_id", task.NodeID),
		zap.String("task_id", task.ID),
		zap.String("execution_id", exec.ID),
		zap.String("trace_id", traceID),
	)

	// Update last run timestamp
	_ = s.taskRepo.UpdateLastRun(ctx, task.ID, &now)

	return exec, nil
}

// HandleExecutionCallback processes the shell_exec result callback from the engine.
func (s *EdgeNodeScheduledTaskService) HandleExecutionCallback(ctx context.Context, nodeID string, req dto.TaskExecutionCallbackRequest) error {
	// Find execution record
	exec, err := s.execRepo.FindByID(ctx, req.TaskID)
	if err != nil {
		return fmt.Errorf("执行记录不存在: %w", err)
	}

	if exec.Status != model.TaskExecRunning {
		// Already processed or timed out — log and skip
		zap.L().Warn("execution callback received for non-running record",
			zap.String("execution_id", req.TaskID),
			zap.String("current_status", exec.Status),
		)
		return nil
	}

	status := req.Status
	if req.ExitCode != 0 && status == model.TaskExecSuccess {
		// If exit code non-zero but status reported as success, treat as failed
		status = model.TaskExecFailed
	}

	finishedAt := time.Now()
	if err := s.execRepo.UpdateResult(ctx, exec.ID, status, req.Stdout, req.Stderr, req.ExitCode, finishedAt); err != nil {
		return fmt.Errorf("更新执行结果失败: %w", err)
	}

	zap.L().Info("shell_exec result received",
		zap.String("execution_id", req.TaskID),
		zap.String("status", status),
		zap.Int("exit_code", req.ExitCode),
		zap.Int64("duration_ms", req.DurationMs),
	)

	return nil
}

// HandleShellExecResult processes MQTT shell_exec_result messages from the engine.
func (s *EdgeNodeScheduledTaskService) HandleShellExecResult(ctx context.Context, nodeID string, payload map[string]interface{}) {
	executionID, _ := payload["execution_id"].(string)
	statusStr, _ := payload["status"].(string)
	stdout, _ := payload["stdout"].(string)
	stderr, _ := payload["stderr"].(string)
	exitCodeFloat, _ := payload["exit_code"].(float64)
	// durationMs not stored directly, but we have finished_at

	if executionID == "" {
		zap.L().Warn("shell_exec result missing execution_id", zap.String("node_id", nodeID))
		return
	}

	status := model.TaskExecFailed
	switch statusStr {
	case "success":
		status = model.TaskExecSuccess
	case "failed":
		status = model.TaskExecFailed
	case "timeout":
		status = model.TaskExecTimeout
	}

	if exitCodeFloat != 0 && status == model.TaskExecSuccess {
		status = model.TaskExecFailed
	}

	exitCode := int(exitCodeFloat)
	finishedAt := time.Now()

	if err := s.execRepo.UpdateResult(ctx, executionID, status, stdout, stderr, exitCode, finishedAt); err != nil {
		zap.L().Error("failed to update execution result from MQTT",
			zap.String("execution_id", executionID),
			zap.Error(err),
		)
	}
}

// TriggerCronTasks evaluates all active cron tasks and executes any that are due.
func (s *EdgeNodeScheduledTaskService) TriggerCronTasks(ctx context.Context) {
	tasks, err := s.taskRepo.ListActiveCron(ctx)
	if err != nil {
		zap.L().Error("failed to list active cron tasks", zap.Error(err))
		return
	}

	now := time.Now()
	for _, task := range tasks {
		// Simple cron matching: if last_run_at is more than 1 minute ago and
		// cron expression is present, execute.
		// In production, use a proper cron library like robfig/cron.
		if task.CronExpr == "" {
			continue
		}

		if task.LastRunAt == nil || now.Sub(*task.LastRunAt) >= time.Minute {
			zap.L().Info("triggering cron task execution",
				zap.String("task_id", task.ID),
				zap.String("name", task.Name),
				zap.String("node_id", task.NodeID),
			)

			if _, err := s.ExecuteNow(ctx, &task); err != nil {
				zap.L().Error("failed to execute cron task",
					zap.String("task_id", task.ID),
					zap.Error(err),
				)
			}
		}
	}
}

// TriggerOneShotTasks evaluates pending one-shot tasks and executes them.
func (s *EdgeNodeScheduledTaskService) TriggerOneShotTasks(ctx context.Context, nodeID string) {
	// This can be called when a node comes online to dispatch pending one-shot tasks.
	tasks, err := s.taskRepo.ListOneShotPending(ctx, nodeID)
	if err != nil {
		zap.L().Error("failed to list one-shot pending tasks",
			zap.String("node_id", nodeID),
			zap.Error(err),
		)
		return
	}

	for _, task := range tasks {
		zap.L().Info("triggering one-shot task execution",
			zap.String("task_id", task.ID),
			zap.String("name", task.Name),
			zap.String("node_id", nodeID),
		)

		if _, err := s.ExecuteNow(ctx, &task); err != nil {
			zap.L().Error("failed to execute one-shot task",
				zap.String("task_id", task.ID),
				zap.Error(err),
			)
		}
	}
}
