# Task Suspension & Recovery

## Overview
When an edge node goes offline, running tasks on that node should be suspended. When the node comes back online, those tasks should be automatically recovered. This document maps the current implementation.

---

## AIVisionTask Model

### File
`/Users/niko/dev/go/aivisioninference/app/internal/model/ai_vision_task.go`

### Status Constants
```go
TaskStatusDraft     = "draft"
TaskStatusReady     = "ready"
TaskStatusRunning   = "running"
TaskStatusError     = "error"
TaskStatusStopped   = "stopped"
TaskStatusSuspended = "suspended"
```

### Key Fields
```go
type AIVisionTask struct {
    BaseModel
    Name            string         // Task name
    Status          string         // draft/ready/running/error/stopped/suspended
    ScheduleID      string         // Time schedule config ID
    DeviceChannelID string         // GB28181 channel ID
    AlgoPackageID   string         // Algorithm package ID
    TargetNodeID    string         // Target edge node ID
    StartDate       datatypes.Date // Effective start date
    EndDate         datatypes.Date // Effective end date
    TimeWindows     datatypes.JSON // Daily time windows
    AIParams        datatypes.JSON // AI parameters
    ROIRegions      datatypes.JSON // ROI regions
    MarkRegions     datatypes.JSON // Mark regions
    LineRegions     datatypes.JSON // Line regions
    ErrorReason     string         // Error/suspension reason
}
```

---

## Task Repository (Suspension/Recovery Methods)

### File
`/Users/niko/dev/go/aivisioninference/app/internal/repository/aivisiontask.go`

### Key Methods

| Method | What it does | SQL Effect |
|--------|-------------|------------|
| `FindRunningTasksByNode(ctx, nodeID)` | Get all running tasks on a node | `WHERE target_node_id = ? AND status = 'running'` |
| `FindSuspendedTasksByNode(ctx, nodeID)` | Get all suspended tasks on a node | `WHERE target_node_id = ? AND status = 'suspended'` |
| `SetErrorReason(ctx, taskID, reason)` | Suspend a task with reason | `UPDATE SET status='suspended', error_reason=reason` |
| `ClearErrorReason(ctx, taskID)` | Resume a task, clear error | `UPDATE SET status='running', error_reason=''` |
| `FindActiveByNode(ctx, nodeID, excludeID)` | Find ready/running tasks | `WHERE target_node_id = ? AND status IN ('ready','running')` |

---

## Task Suspension Flow

### Trigger 1: Periodic Liveness Check
**File:** `/Users/niko/dev/go/aivisioninference/app/internal/task/edge_node_status.go`

**But this task NEVER RUNS** (not registered as periodic schedule — see research/03).

### Trigger 2: MQTT LWT (HandleLifecycle)
**File:** `/Users/niko/dev/go/aivisioninference/app/internal/handler/edge_mqtt_handler.go`

**But this handler does NOT suspend tasks** — it only sets node status to "offline".

### Current Suspension Gap
Tasks are **never automatically suspended** when a node goes offline. The code for suspension exists (`suspendNodeTasks`), but:
- The periodic check is not scheduled
- The LWT handler doesn't call it

### What `suspendNodeTasks` does (if it were invoked):
```go
func (h *EdgeNodeStatusTask) suspendNodeTasks(ctx context.Context, node model.EdgeNode) {
    runningTasks, err := h.taskRepo.FindRunningTasksByNode(ctx, node.ID)
    for _, task := range runningTasks {
        reason := fmt.Sprintf("节点 %s 离线，任务自动暂停", node.Name)
        h.taskRepo.SetErrorReason(ctx, task.ID, reason) // → status='suspended', error_reason=reason
        
        // Broadcast WebSocket event
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
```

---

## Task Recovery Flow

### Trigger: Heartbeat from previously-offline node
**File:** `/Users/niko/dev/go/aivisioninference/app/internal/service/edge_node.go` — `HandleHeartbeat()`

When a node comes back online (status determined to be "online"):
```go
if status == model.NodeStatusOnline {
    tasks, err := s.taskRepo.FindSuspendedTasksByNode(ctx, id)
    for _, task := range suspendedTasks {
        s.taskRepo.ClearErrorReason(ctx, task.ID) // → status='running', error_reason=''
        
        // Broadcast WebSocket event
        s.hub.Broadcast(&ws.Message{
            Type: "task-status",
            Payload: map[string]interface{}{
                "task_id":      task.ID,
                "status":       model.TaskStatusRunning,
                "error_reason": "",
                "node_id":      id,
            },
        })
    }
}
```

### Recovery Scope
- Only tasks in `"suspended"` status for that node are recovered
- Tasks that were in `"running"` status but never suspended (because suspension didn't fire) are NOT affected — they remain in "running" limbo
- The `AIVisionTaskPatrol` (`* * * * *`) handles time-window-based start/stop, not node-based recovery

---

## Task Status Lifecycle

```
draft ──→ ready ──→ running ──→ stopped
                        │
                        ├──→ error
                        │
                        └──→ suspended ──→ running (on node recovery)
```

---

## State Reconciliation

### File
`/Users/niko/dev/go/aivisioninference/app/internal/task/edge_worker.go`

### Task Type
```go
const TaskReconcileEdgeState = "edge_node:reconcile_state"
```

### Trigger
Enqueued by MQTT heartbeat handler when active streams hash changes.

### What it does:
1. Fetches `running` tasks from DB for the node → `expectedDeviceIDs`
2. Fetches `actual_streams` from Redis (stored by MQTT heartbeat handler)
3. For any actual stream NOT in expected set → calls `engineClient.StopStream()` to kill ghost streams

### What it does NOT do:
- Does NOT start missing streams (tasks in "suspended" that should be "running")
- Does NOT handle nodes that disappeared entirely
- Only kills ghost streams on nodes that are still heartbeating

---

## Task Patrol

### File
`/Users/niko/dev/go/aivisioninference/app/internal/task/aivisiontask_patrol.go`

### Type
```go
const TypeAIVisionTaskPatrol = "aivision:patrol"
```

### Frequency: Every minute (`* * * * *`)

### Purpose: Time-window-based task start/stop
- Runs `AIVisionTaskService.PatrolTasks()`
- Checks if running tasks should be stopped (outside time window)
- Checks if ready tasks should be started (inside time window)
- Does NOT handle node-level recovery

---

## What's Missing / Broken

1. **❌ Node suspend → task suspend gap** — No code path reliably sets tasks to "suspended" when a node goes offline. The `suspendNodeTasks` logic exists but is unreachable.

2. **❌ LWT handler does not suspend tasks** — The `HandleLifecycle` handler sets node status to "offline" but leaves tasks in "running" limbo.

3. **No task re-scheduling** — When a node goes offline, tasks are just suspended. There's no mechanism to re-assign them to another online node.

4. **No "manual suspend" API** — No HTTP endpoint to manually suspend/resume tasks.

5. **Recovery only works for tasks that were properly suspended** — If a task was left in "running" state when the node died, the recovery flow won't find it (it only looks for "suspended").

6. **No task pre-emption check** — When recovering suspended tasks, the system doesn't verify the node has capacity (CurrentLoad vs MaxLoad) before resuming.

7. **No recovery for tasks stuck in "running" state** — If a task was never suspended, it stays "running" forever on a dead node. There's no reconciliation for this.
