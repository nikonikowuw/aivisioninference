# 修复节点离线检测与任务恢复 — Implementation Plan

## Overview

Implement unified node offline detection, MQTT LWT convergence, and Engine-verified task recovery for edge nodes.

## Step-by-Step Checklist

### Step 1: Add `suspended_reason` to AIVisionTask model

**Files:**
- `app/internal/model/ai_vision_task.go`

**Changes:**
- Add `SuspendedReason` field with `*string` type, GORM column `suspended_reason`
- Add constants: `SuspendedReasonNodeOffline = "node_offline"`, `SuspendedReasonManual = "manual"`, `SuspendedReasonSchedule = "schedule"`, `SuspendedReasonError = "error"`

**Verify:** `cd app && go build ./internal/model/`

**Target:** Step 1 complete

---

### Step 2: Add DB migration for `suspended_reason`

**Files:**
- `app/cmd/migrate/main.go` or migration file

**Changes:**
- Add `ALTER TABLE ai_vision_tasks ADD COLUMN IF NOT EXISTS suspended_reason VARCHAR(50);`
- Add `CREATE INDEX IF NOT EXISTS idx_ai_vision_tasks_suspended_reason ON ai_vision_tasks(suspended_reason);`

**Verify:** Run `go run app/cmd/migrate/main.go` (against test DB)

**Target:** Step 2 complete

---

### Step 3: Add `WithTx` to AIVisionTaskRepository

**Files:**
- `app/internal/repository/aivisiontask.go`

**Changes:**
- Add `WithTx(tx *gorm.DB) *AIVisionTaskRepository` method

**Verify:** `cd app && go build ./internal/repository/`

**Target:** Step 3 complete

---

### Step 4: Update AIVisionTaskRepository suspend/resume methods

**Files:**
- `app/internal/repository/aivisiontask.go`

**Changes:**
- Replace `SetErrorReason` -> `SetSuspended(ctx, taskID, reason, errorMsg)`:
  - Sets status='suspended', suspended_reason=reason, error_reason=errorMsg
- Replace `ClearErrorReason` -> `ClearSuspended(ctx, taskID)`:
  - Sets status='running', suspended_reason=NULL, error_reason=''
- Add `FindNodeOfflineSuspendedTasks(ctx, nodeID)`: filters suspended tasks by node AND `suspended_reason = 'node_offline'`

**Verify:** `cd app && go build ./internal/repository/` && `cd app && go test ./internal/repository/ -run TestAIVisionTask`

**Target:** Step 4 complete

---

### Step 5: Add `DB()` accessor to EdgeNodeRepository

**Files:**
- `app/internal/repository/edge_node.go`

**Changes:**
- Add `DB(ctx context.Context) *gorm.DB` method that returns the underlying DB for service-level transaction creation

**Verify:** `cd app && go build ./internal/repository/`

**Target:** Step 5 complete

---

### Step 6: Create shared `HandleNodeOffline` function

**Files:**
- `app/internal/task/edge_node_liveness.go` (new file)

**Changes:**
- Create a standalone function (or method) that encapsulates the offline handling logic:
  - Mark node status = offline (batch update)
  - Suspend all running tasks with `suspended_reason = 'node_offline'`
  - Broadcast WebSocket edge-node-status = offline
  - Broadcast WebSocket task-status = suspended per task
- This function is called from both `handleEdgeNodeStatusCheck` and `HandleLifecycle`

```go
// edge_node_liveness.go (new)
package task

func HandleNodeOffline(
    ctx context.Context,
    nodeRepo *repository.EdgeNodeRepository,
    taskRepo *repository.AIVisionTaskRepository,
    hub *ws.Hub,
    node model.EdgeNode,
    suspendReason string,  // model.SuspendedReasonNodeOffline
    reasonFmt string,      // "节点 %s 离线%s，任务自动暂停"
    extra ...interface{},
)
```

**Verify:** `cd app && go build ./internal/task/`

**Target:** Step 6 complete

---

### Step 7: Refactor EdgeNodeStatusTask to use shared offline handler

**Files:**
- `app/internal/task/edge_node_status.go`

**Changes:**
- In `handleEdgeNodeStatusCheck`:
  - Keep `FindTimedOutNodes` and `UpdateStatusBatch`
  - Replace inline suspend logic with call to `HandleNodeOffline` for each timed-out node
  - Use `SuspendedReasonNodeOffline` as the suspend reason

**Verify:** `cd app && go test ./internal/task/ -run TestEdgeNodeStatusTask`

**Target:** Step 7 complete

---

### Step 8: Refactor HandleLifecycle to use shared offline handler

**Files:**
- `app/internal/handler/edge_mqtt_handler.go`

**Changes:**
- In `HandleLifecycle`, when payload indicates disconnect/offline:
  - Call `HandleNodeOffline` from the task package
  - Need to inject `EdgeNodeStatusTask` or the required repos into `EdgeMqttHandler`

Design note: `EdgeMqttHandler` currently doesn't have `nodeRepo`/`taskRepo`/`hub`. Options:
1. Add dependencies to `EdgeMqttHandler` constructor
2. Create a thin `LivenessHandler` interface and inject into EdgeMqttHandler
3. Let EdgeMqttHandler call the service layer

Chosen approach: Option 1 — add nodeRepo, taskRepo, hub to EdgeMqttHandler.

**Files to update:**
- `app/internal/handler/edge_mqtt_handler.go` — add fields and constructor params
- `app/internal/router/deps.go` — update provider function to pass new deps
- `app/internal/router/wire.go` — update wiring (if auto-wired)

**Verify:** `cd app && go build ./internal/handler/` && `cd app && make wire`

**Target:** Step 8 complete

---

### Step 9: Fix `remark` overwrite in HandleHeartbeat

**Files:**
- `app/internal/service/edge_node.go`

**Changes:**
- Remove `remark = req.ErrorMessage` assignment
- If `req.Status == "error"`, store error message in a separate field or log it without overwriting remark
- Keep `remark` out of `hbFields` map when it's just an error message
- Consider adding `last_error` field to EdgeNode model for storing engine-reported errors (optional — if this causes too much scope creep, just stop overwriting remark)

**Minimal change:** Just remove the remark assignment. The `remark` field stays unchanged from its current DB value.

```go
// Before:
remark := ""
if req.Status == "error" {
    status = model.NodeStatusError
    remark = req.ErrorMessage
} else if node.Status == model.NodeStatusDisabled {
    status = model.NodeStatusDisabled
}

// After:
// Don't overwrite remark. Store error separately.
if req.Status == "error" {
    status = model.NodeStatusError
    // Log the error but don't overwrite remark
    zap.L().Info("node reported error", 
        zap.String("node_id", id), 
        zap.String("error_message", req.ErrorMessage))
} else if node.Status == model.NodeStatusDisabled {
    status = model.NodeStatusDisabled
}
```

Also remove `"remark": remark,` from hbFields map when remark is empty.

**Verify:** `cd app && go build ./internal/service/` && `cd app && go test ./internal/service/ -run TestEdgeNodeService`

**Target:** Step 9 complete

---

### Step 10: Add disabled node guard in HandleHeartbeat

**Files:**
- `app/internal/service/edge_node.go`

**Changes:**
- After loading node and checking version compatibility, add guard:
```go
if node.Status == model.NodeStatusDisabled || !node.Enabled {
    // Still update last_heartbeat so admin can see liveness, but skip all processing
    now := time.Now()
    hbFields := map[string]interface{}{
        "last_heartbeat": &now,
    }
    s.nodeRepo.UpdateHeartbeatFields(ctx, id, hbFields)
    return &dto.HeartbeatResponse{}, nil
}
```

**Verify:** `cd app && go build ./internal/service/` && test

**Target:** Step 10 complete

---

### Step 11: Rework HandleHeartbeat recovery with Engine ACK

**Files:**
- `app/internal/service/edge_node.go`

**Changes:**
After the transaction commits and the node is back online, instead of blindly clearing all suspended tasks:
1. Query tasks with `suspended_reason = 'node_offline'` 
2. For each task, call EngineClient.StartStream (or StartPlayback) with timeout
3. If Engine ACKs, ClearSuspended the task
4. If Engine fails, keep suspended and set error

```go
// After transaction commits
if recoveredToOnline {
    tasks, err := s.taskRepo.FindNodeOfflineSuspendedTasks(ctx, id)
    for _, task := range tasks {
        err = s.restoreTaskPipeline(ctx, node, task)
        if err != nil {
            zap.L().Error("failed to restore task pipeline",
                zap.String("task_id", task.ID),
                zap.String("node_id", id),
                zap.Error(err))
            // Keep task suspended
        }
    }
}
```

Add helper `restoreTaskPipeline`:
```go
func (s *EdgeNodeService) restoreTaskPipeline(ctx context.Context, node *model.EdgeNode, task model.AIVisionTask) error {
    deviceID := task.DeviceChannelID
    if deviceID == "" {
        return fmt.Errorf("task %s has no device_channel_id, cannot restore", task.ID)
    }
    
    startCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
    defer cancel()
    
    if err := s.engineClient.StartStream(startCtx, node.ID, deviceID, task.AIParams); err != nil {
        return fmt.Errorf("engine start_stream failed: %w", err)
    }
    
    return s.taskRepo.ClearSuspended(ctx, task.ID)
}
```

Note: This requires `EdgeNodeService` to have an `engineClient` field. Currently it doesn't. Need to add `EngineClient` to `EdgeNodeService` constructor.

**Files to update:**
- `app/internal/service/edge_node.go` — add EngineClient field, update constructor and call sites
- `app/internal/router/deps.go` — update `provideEdgeNodeService` to pass engineClient
- `app/internal/router/wire.go` — update wiring (if auto-wired)

**Verify:** `cd app && go build ./internal/service/` && `cd app && make wire`

**Target:** Step 11 complete

---

### Step 12: Register periodic edge_node:status_check schedule

**Files:**
- `app/internal/task/edge_node_status.go` — add RegisterPeriodic method
- `app/internal/server/server.go` — update Run() to call it
- OR: `app/internal/router/router.go` — call in NewAsynqMux

**Changes:**
Add self-registration method on EdgeNodeStatusTask:

```go
// In edge_node_status.go
func (h *EdgeNodeStatusTask) RegisterPeriodic(scheduler *asynq.Scheduler, intervalSec int) {
    if intervalSec <= 0 {
        zap.L().Warn("edge node status check interval not configured, skipping periodic registration")
        return
    }
    scheduler.Register(fmt.Sprintf("@every %ds", intervalSec),
        asynq.NewTask(TypeEdgeNodeStatusCheck, nil))
    zap.L().Info("registered periodic edge node status check",
        zap.Int("interval_sec", intervalSec))
}
```

Call in `NewAsynqMux` in `router.go`:
```go
// After creating edgeNodeStatusTask:
edgeNodeStatusTask.RegisterPeriodic(scheduler, cfg.Engine.HeartbeatCheckIntervalSec)
```

Note: Need to pass `*asynq.Scheduler` to `NewAsynqMux` or wire it in the server startup.

Currently `NewAsynqMux` doesn't receive the scheduler. Options:
1. Pass scheduler to NewAsynqMux — requires changing router.go
2. Register in server.go after getting scheduler — simpler

Chosen: Register in `NewAsynqMux` since that's where the task handler is created. Pass scheduler as parameter.

**Verify:** `cd app && go build ./internal/router/` && check the periodic task appears in scheduler registration

**Target:** Step 12 complete

---

### Step 13: Verify EngineClient interface supports required methods

**Files:**
- `app/internal/service/ipc_engine_client.go` — interface definition
- `app/internal/service/mqtt_engine_client.go` — implementation

**Check:**
- `StartStream(ctx, nodeID, deviceID, params) error` — should accept AI params or stream settings
- Ensure timeout handling is in place (MQTT sync wait)
- `StartPlayback(ctx, nodeID, deviceID, params) error` — similar

**No changes needed if interface already covers the use case.**

**Verify:** Read interface and confirm method signatures match what `restoreTaskPipeline` needs.

**Target:** Step 13 complete

---

### Step 14: Add EngineClient dependency to EdgeNodeService

**Files:**
- `app/internal/service/edge_node.go`

**Changes:**
- Add `engineClient service.EngineClient` field to `EdgeNodeService` struct
- Update constructor `NewEdgeNodeService` to accept `engineClient service.EngineClient`
- Update all call sites (deps.go, wire.go)

**Verify:** `cd app && go build ./internal/service/` && `cd app && make wire`

**Target:** Step 14 complete

---

### Step 15: Unit tests

**Files:**
- `app/internal/task/edge_node_status_test.go` — extend existing tests
- `app/internal/service/edge_node_test.go` — add heartbeat/recovery tests
- `app/internal/repository/aivisiontask_test.go` — add suspend/resume tests

**Test cases:**
1. `handleEdgeNodeStatusCheck` finds timed-out nodes and marks offline
2. Suspended tasks have correct `suspended_reason` after offline
3. Heartbeat recovery only restores `suspended_reason='node_offline'` tasks
4. Heartbeat recovery skips tasks with other suspended_reason
5. Heartbeat recovery calls EngineClient and handles failure gracefully
6. Disabled node heartbeat returns early without processing
7. Remark is NOT overwritten during error heartbeat
8. LWT handler invokes same offline logic as periodic check

**Verify:** `cd app && go test ./internal/... -count=1`

**Target:** Step 15 complete

---

### Step 16: Integration/regression tests

**Additional test considerations:**
- Test that the periodic schedule is actually registered (integration test)
- Test async scenario: node times out → offline → heartbeat → recovery with Engine ACK
- Test that MQTT LWT message triggers offline handling

**Verify:** `cd app && go test ./internal/... -count=1 -run TestEdge`

**Target:** Step 16 complete

---

### Step 17: Re-generate Wire

```bash
cd app && make wire
```

**Verify:** `cd app && make build`

**Target:** Step 17 complete

---

### Step 18: Acceptance Criteria Verification

| # | Criteria | How to verify |
|---|----------|---------------|
| 1 | Periodic scheduler delivers edge node status check | Check scheduler logs for `TypeEdgeNodeStatusCheck` dispatch |
| 2 | MQTT LWT and heartbeat timeout produce consistent offline results | Unit test coverage for both paths calling shared handler |
| 3 | Running tasks on offline node marked as node-offline suspended | Check `suspended_reason` in DB after offline |
| 4 | Non-node-offline suspended tasks not auto-recovered | Test verifies heartbeat skips manual/error suspended tasks |
| 5 | Engine pipeline recreated before task marked running | Test mocks EngineClient, verifies StartStream called before ClearSuspended |
| 6 | Offline, re-offline, recovery-success, recovery-failure all tested | Unit coverage in edge_node_test.go and edge_node_status_test.go |

**Target:** Step 18 complete — ready for review

---

## Quick Reference

| Step | File(s) | Action |
|------|---------|--------|
| 1 | `model/ai_vision_task.go` | Add `suspended_reason` field + constants |
| 2 | Migration | Add column + index |
| 3 | `repository/aivisiontask.go` | Add `WithTx` method |
| 4 | `repository/aivisiontask.go` | New `SetSuspended`/`ClearSuspended`/`FindNodeOfflineSuspendedTasks` |
| 5 | `repository/edge_node.go` | Add `DB()` accessor |
| 6 | `task/edge_node_liveness.go` (new) | Create shared `HandleNodeOffline` |
| 7 | `task/edge_node_status.go` | Refactor to use shared handler |
| 8 | `handler/edge_mqtt_handler.go`, `router/deps.go` | Wire LWT to shared handler |
| 9 | `service/edge_node.go` | Remove remark overwrite |
| 10 | `service/edge_node.go` | Add disabled node guard |
| 11 | `service/edge_node.go`, `router/deps.go` | Engine ACK recovery |
| 12 | `task/edge_node_status.go`, `router/router.go` | Register periodic schedule |
| 13 | `service/ipc_engine_client.go` | Verify EngineClient interface |
| 14 | `service/edge_node.go`, `router/deps.go`, `router/wire.go` | Add EngineClient to EdgeNodeService |
| 15 | Test files | Unit tests for all changes |
| 16 | Test files | Integration tests |
| 17 | — | `cd app && make wire && make build` |
