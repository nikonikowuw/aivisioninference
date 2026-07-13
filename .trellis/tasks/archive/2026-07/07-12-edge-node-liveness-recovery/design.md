# 修复节点离线检测与任务恢复 — Design

## 1. Overview

This document describes the technical design for fixing edge node offline detection and task recovery in the control plane. The goal is to establish a unified, configurable, idempotent offline/online lifecycle that guarantees task state accurately reflects the Engine Pipeline.

### Current State

- `EdgeNodeStatusTask.handleEdgeNodeStatusCheck` IS implemented — finds timed-out online nodes, marks them offline, suspends running tasks.
- But the periodic schedule `TypeEdgeNodeStatusCheck` is **NOT registered** in `RegisterPeriodicTasks` (`task/server.go`), so **no periodic check ever executes**.
- MQTT LWT topic is published by Engine (`aivision/edge/{node_id}/status/lifecycle`) and handled by `HandleLifecycle` in `edge_mqtt_handler.go`, but there is no convergence with HTTP timeout handling.
- Heartbeat recovery (`HandleHeartbeat` in `service/edge_node.go`) calls `ClearErrorReason` which blindly sets all suspended tasks to `running` without Engine confirmation.
- No `suspended_reason` field exists on `AIVisionTask` — cannot distinguish node-offline suspension from manual/error suspension.
- `remark` field is overwritten with error message when node reports error status.
- Transaction in heartbeat processing doesn't propagate the tx handle to sub-repos (`AIVisionTaskRepository`, `EdgeNodeAlgorithmRepository`).

## 2. Changes

### 2.1 Model Changes

**AIVisionTask — add `suspended_reason` field**

Add a nullable string field `suspended_reason` to `AIVisionTask`:

```go
// SuspendedReason constants
const (
    SuspendedReasonNodeOffline = "node_offline"
    SuspendedReasonManual      = "manual"
    SuspendedReasonSchedule    = "schedule"
    SuspendedReasonError       = "error"
)

// Add to AIVisionTask struct:
SuspendedReason *string `gorm:"type:varchar(50);index;comment:暂停原因(node_offline/manual/schedule/error)" json:"suspended_reason,omitempty"`
```

This field is set when a task is suspended and checked during auto-recovery. Only tasks with `suspended_reason = 'node_offline'` are auto-resumed.

No data migration needed for existing records — `nil` suspended_reason means "legacy/unknown" and will NOT be auto-resumed.

### 2.2 Repository Changes

**AIVisionTaskRepository — add methods for suspend source tracking**

- `SetSuspended(ctx, taskID, reason, errorMsg)` — atomically sets status='suspended', suspended_reason, error_reason
- `ClearSuspended(ctx, taskID)` — clears status→'running', suspended_reason→nil, error_reason→''
- `FindSuspendedTasksByNodeAndReason(ctx, nodeID, reason)` — same as FindSuspendedTasksByNode but filtered by reason
- `FindNodeOfflineSuspendedTasks(ctx, nodeID)` — convenience that calls FindSuspendedTasksByNodeAndReason with reason='node_offline'
- `WithTx(tx)` — add WithTx method to AIVisionTaskRepository for transaction propagation

**EdgeNodeRepository — add DB() accessor**

Add `DB(ctx context.Context) *gorm.DB` to allow service-level transaction creation.

**EdgeNodeAlgorithmRepository — verify WithTx exists, add if missing**

### 2.3 Service Changes

**EdgeNodeService.HandleHeartbeat — rework recovery logic**

Current logic:
```go
// Inside transaction:
if status == model.NodeStatusOnline {
    tasks, err := s.taskRepo.FindSuspendedTasksByNode(ctx, id)
    for _, task := range tasks {
        s.taskRepo.ClearErrorReason(ctx, task.ID)
    }
}
```

New logic:
```go
// After transaction commits successfully:
if node came back online (status transitioned to online):
    for each task suspended with reason 'node_offline':
        // Send Engine command to restart pipeline
        if task has DeviceChannelID:
            err = engineClient.StartStream(ctx, nodeID, task.DeviceChannelID, task.AIParams)
        else:
            err = engineClient.StartPlayback(ctx, nodeID, task.DeviceChannelID, playbackParams)
        
        if err == nil:
            taskRepo.ClearSuspended(ctx, task.ID)  // Mark running
        else:
            taskRepo.UpdateError(ctx, task.ID, "Engine pipeline restart failed: " + err.Error())
            // Keep suspended — don't clear
```

Note: Engine commands are sent AFTER the DB transaction commits. This avoids the risk of Engine ACK but DB rollback (or vice versa). If Engine command fails, the task stays `suspended` with an error message.

**EdgeNodeService.HandleHeartbeat — fix remark overwrite**

Remove `remark = req.ErrorMessage` assignment. Use a separate field for Engine-reported errors (e.g., a dedicated `last_error` field on `EdgeNode` model), or log the error without overwriting `remark`.

**EdgeNodeService.HandleHeartbeat — add disabled node guard**

After loading the node, check `!node.Enabled` or `node.Status == NodeStatusDisabled` and return early:
```go
if !node.Enabled || node.Status == model.NodeStatusDisabled {
    zap.L().Warn("heartbeat from disabled node, ignoring", zap.String("node_id", id))
    // Still update last_heartbeat to track liveness but don't process
    return &dto.HeartbeatResponse{}, nil
}
```

### 2.4 Task Changes

**RegisterPeriodicTasks — add edge_node:status_check schedule**

In `task/server.go`, add:
```go
// Edge node heartbeat timeout check (every heartbeat_check_interval_sec)
if cfg.Engine.HeartbeatCheckIntervalSec > 0 {
    scheduler.Register(fmt.Sprintf("@every %ds", cfg.Engine.HeartbeatCheckIntervalSec),
        asynq.NewTask(TypeEdgeNodeStatusCheck, nil))
}
```

This requires passing the engine config to `RegisterPeriodicTasks`, or having `EdgeNodeStatusTask` self-register its schedule. 

Design decision: Keep registration in `RegisterPeriodicTasks` but change its signature to accept engine config. OR, create a new method `EdgeNodeStatusTask.RegisterPeriodic(scheduler, intervalSec)` that self-registers.

Chosen approach: Add a self-registration method to `EdgeNodeStatusTask` for clean separation:
```go
func (h *EdgeNodeStatusTask) RegisterPeriodic(scheduler *asynq.Scheduler, intervalSec int) {
    if intervalSec <= 0 {
        return
    }
    scheduler.Register(fmt.Sprintf("@every %ds", intervalSec),
        asynq.NewTask(TypeEdgeNodeStatusCheck, nil))
}
```

Call from `NewAsynqMux` where EdgeNodeStatusTask is created.

**EdgeNodeStatusTask.handleEdgeNodeStatusCheck — use suspended_reason**

When suspending tasks after marking node offline, set `suspended_reason = 'node_offline'`:
```go
func (h *EdgeNodeStatusTask) suspendNodeTasks(ctx context.Context, node model.EdgeNode) {
    for _, task := range runningTasks {
        h.taskRepo.SetSuspended(ctx, task.ID, 
            model.SuspendedReasonNodeOffline, 
            fmt.Sprintf("节点 %s 离线，任务自动暂停", node.Name))
    }
}
```

### 2.5 MQTT LWT Convergence

**HandleLifecycle handler — reuse EdgeNodeStatusTask offline logic**

Currently `HandleLifecycle` in `edge_mqtt_handler.go` subscribes to `aivision/edge/+/status/lifecycle` but its logic is separate from `handleEdgeNodeStatusCheck`.

Design: Create a shared `NodeOfflineHandler` interface or function that can be called from both:
- `EdgeNodeStatusTask.handleEdgeNodeStatusCheck` (periodic timeout)
- `HandleLifecycle` (MQTT LWT message)

The shared function:
1. If payload indicates offline/disconnect:
   - Mark node as offline
   - Suspend running tasks with `suspended_reason = 'node_offline'`
   - Broadcast WebSocket notification
2. If payload indicates online/connect:
   - (Handled by HTTP heartbeat, not LWT)

```go
// Shared function in a separate file, e.g., task/edge_node_liveness.go
func HandleNodeOffline(ctx context.Context, nodeRepo, taskRepo, hub, nodeID) {
    // 1. Mark node offline
    nodeRepo.UpdateStatusBatch(ctx, []string{nodeID}, model.NodeStatusOffline)
    
    // 2. Suspend tasks
    tasks, _ := taskRepo.FindRunningTasksByNode(ctx, nodeID)
    for _, task := range tasks {
        taskRepo.SetSuspended(ctx, task.ID, 
            model.SuspendedReasonNodeOffline,
            fmt.Sprintf("节点 %s 离线(MQTT LWT)，任务自动暂停", nodeID))
    }
    
    // 3. WebSocket notification
    hub.Broadcast(...)
}
```

Then `HandleLifecycle` in `edge_mqtt_handler.go` calls this function, and `EdgeNodeStatusTask.handleEdgeNodeStatusCheck` also uses it per-node.

## 3. Data Flow

### 3.1 Normal Node Offline (Heartbeat Timeout)

```text
Asynq Scheduler (every heartbeat_check_interval_sec)
  → edge_node:status_check task
  → EdgeNodeStatusTask.handleEdgeNodeStatusCheck
  → nodeRepo.FindTimedOutNodes(cutoff)          # Find online nodes with stale heartbeat
  → For each timed-out node:
      → nodeRepo.UpdateStatusBatch → offline    # Batch DB update
      → taskRepo.SetSuspended(reason=node_offline)  # Suspend with source
      → hub.Broadcast(edge-node-status=offline)     # WS notification
      → hub.Broadcast(task-status=suspended)         # Per-task WS notification
```

### 3.2 Node Offline via MQTT LWT

```text
Engine disconnect → MQTT broker publishes LWT
  → MqttMux → HandleLifecycle
  → Parse payload from LWT message
  → Call HandleNodeOffline(nodeID)
      → Same flow as 3.1 (shared function)
```

### 3.3 Node Recovery (Heartbeat Received)

```text
HTTP POST /api/v1/edge-nodes/:id/heartbeat
  → EdgeNodeHandler.Heartbeat
  → EdgeNodeService.HandleHeartbeat
      → nodeRepo.Transaction (atomic heartbeat fields + algo sync)
          → UpdateHeartbeatFields(nodeID, hbFields)
          → SyncInstalled(nodeID, installedAlgos)
      → (transaction commits)
      → For each task with suspended_reason = 'node_offline':
          → SendEngineCommand(StartStream/StartPlayback)  # MQTT with ACK wait
          → If Engine confirms:
              → taskRepo.ClearSuspended(taskID)           # Mark running
              → hub.Broadcast(task-status=running)
          → If Engine fails:
              → taskRepo.SetError(taskID, reason)         # Keep suspended
              → hub.Broadcast(task-status=suspended, error)
      → hub.Broadcast(edge-node-status=online)
      → Return heartbeat response with pending deployments
```

### 3.4 Disabled Node Heartbeat

```text
HTTP POST /api/v1/edge-nodes/:id/heartbeat
  → EdgeNodeService.HandleHeartbeat
  → node.Status == 'disabled' OR !node.Enabled
  → Return early (heartbeat accepted for tracking, no actions)

Note: last_heartbeat still updated so admin can see when disabled node last reported.
```

## 4. Compatibility & Migration

- **New `suspended_reason` field**: NULL for existing records. NULL means "legacy/unknown" — these tasks are NOT auto-resumed by the new logic, preserving backward compatibility.
- **Engine pipeline recovery**: Only triggers for newly suspended tasks (those with `suspended_reason='node_offline'`). Existing suspended tasks remain in their current state.
- **Transaction propagation**: Only affects the heartbeat handler path. Other callers maintain existing behavior.

## 5. Rollout Plan

1. Add `suspended_reason` column to `ai_vision_tasks` table (migration)
2. Add `WithTx` to `AIVisionTaskRepository`
3. Add `LastError` field to `EdgeNode` (optional, for storing engine error without overwriting remark)
4. Create shared `HandleNodeOffline` function
5. Refactor `EdgeNodeStatusTask.handleEdgeNodeStatusCheck` to use shared function + suspended_reason
6. Refactor `HandleLifecycle` to call shared function
7. Rework `HandleHeartbeat` recovery with Engine ACK wait
8. Add disabled node guard
9. Register periodic schedule
10. Add unit/integration tests
11. Migration SQL for existing DB

## 6. Out of Scope

- Full transactional propagation across all repos (punted to separate task; this task only fixes the heartbeat path)
- Engine-side changes (LWT is already sent by Engine, no changes needed)
- Frontend changes for displaying suspended_reason (handled by contract-security child task)
