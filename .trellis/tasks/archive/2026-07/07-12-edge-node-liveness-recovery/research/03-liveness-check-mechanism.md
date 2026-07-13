# Liveness Check Mechanism

## Overview
The liveness check runs as an Asynq periodic task that queries for nodes whose `last_heartbeat` is older than the configured timeout, marks them offline, suspends their tasks, and broadcasts WebSocket events.

---

## Key Files

| Component | File Path |
|-----------|-----------|
| Task handler | `/Users/niko/dev/go/aivisioninference/app/internal/task/edge_node_status.go` |
| Asynq setup | `/Users/niko/dev/go/aivisioninference/app/internal/router/router.go` |
| Periodic registration | `/Users/niko/dev/go/aivisioninference/app/internal/task/server.go` |
| Repository query | `/Users/niko/dev/go/aivisioninference/app/internal/repository/edge_node.go` |

---

## Task Type
```go
const TypeEdgeNodeStatusCheck = "edge_node:status_check"  // in edge_node_status.go
```

## Handler Registration
In `router.go` (line 616-618):
```go
edgeNodeStatusTask := task.NewEdgeNodeStatusTask(nodeRepo, aiTaskRepo, hub, cfg.Engine.HeartbeatTimeoutSec)
edgeNodeStatusTask.RegisterHandlers(mux)
```

## ❌ CRITICAL: Periodic Schedule NOT Registered

The `edge_node:status_check` task is **registered as a handler** on the Asynq mux, but **NOT registered as a periodic schedule** in `RegisterPeriodicTasks()`.

### What IS registered (`task/server.go`):
```go
func RegisterPeriodicTasks(scheduler *asynq.Scheduler) {
    scheduler.Register("*/5 * * * *", asynq.NewTask(TypeDeviceStatusCheck, nil))  // device:status_check
    scheduler.Register("* * * * *", asynq.NewTask(TypeAIVisionTaskPatrol, nil))    // aivision:patrol
}
```

The `TypeEdgeNodeStatusCheck` is **completely missing** from `RegisterPeriodicTasks()`. This means:
- The handler can receive manual invocations
- But the periodic timeout check NEVER RUNS automatically
- Edge nodes will stay "online" in the DB forever unless LWT fires

---

## Task Implementation

### Constructor
```go
func NewEdgeNodeStatusTask(
    nodeRepo *repository.EdgeNodeRepository,
    taskRepo *repository.AIVisionTaskRepository,
    hub *ws.Hub,
    timeoutSeconds int,  // from cfg.Engine.HeartbeatTimeoutSec
) *EdgeNodeStatusTask
```

### Handler Logic (`handleEdgeNodeStatusCheck`)
```go
func (h *EdgeNodeStatusTask) handleEdgeNodeStatusCheck(ctx context.Context, t *asynq.Task) error {
    cutoff := time.Now().Add(-time.Duration(h.timeoutSeconds) * time.Second)
    
    // Find online nodes with heartbeat older than cutoff
    nodes, err := h.nodeRepo.FindTimedOutNodes(ctx, cutoff)
    // ... if none, return nil
    
    for _, node := range nodes {
        ids = append(ids, node.ID)
        h.broadcastNodeOffline(node.ID)
        h.suspendNodeTasks(ctx, node)
    }
    
    // Batch update all to offline
    h.nodeRepo.UpdateStatusBatch(ctx, ids, model.NodeStatusOffline)
}
```

### Query: `FindTimedOutNodes`
```go
func (r *EdgeNodeRepository) FindTimedOutNodes(ctx context.Context, cutoff time.Time) ([]model.EdgeNode, error) {
    var items []model.EdgeNode
    err := r.db.WithContext(ctx).
        Where("status = ?", model.NodeStatusOnline).
        Where("last_heartbeat < ?", cutoff).
        Find(&items).Error
    return items, err
}
```

### Task Suspension
```go
func (h *EdgeNodeStatusTask) suspendNodeTasks(ctx context.Context, node model.EdgeNode) {
    runningTasks, _ := h.taskRepo.FindRunningTasksByNode(ctx, node.ID)
    for _, task := range runningTasks {
        reason := fmt.Sprintf("节点 %s 离线，任务自动暂停", node.Name)
        h.taskRepo.SetErrorReason(ctx, task.ID, reason)
        // Broadcast task-status WebSocket event
    }
}
```

### WebSocket Broadcast
```go
h.hub.Broadcast(&ws.Message{
    Type: "edge-node-status",
    Payload: map[string]interface{}{
        "node_id": nodeID,
        "status":  model.NodeStatusOffline,
    },
})
```

---

## Configuration

### File
`/Users/niko/dev/go/aivisioninference/app/internal/config/config.go`

```go
type EngineConfig struct {
    MinCompatibleVersion   string `mapstructure:"min_compatible_version"`
    VersionCheckEnabled    bool   `mapstructure:"version_check_enabled"`
    HeartbeatTimeoutSec    int    `mapstructure:"heartbeat_timeout_sec"`
    HeartbeatCheckInterval int    `mapstructure:"heartbeat_check_interval_sec"`
}
```

### Defaults
```go
v.SetDefault("engine.heartbeat_timeout_sec", 15)
v.SetDefault("engine.heartbeat_check_interval_sec", 10)
```

### Used by
- `HeartbeatTimeoutSec` → passed to `EdgeNodeStatusTask` for `timeoutSeconds`
- `HeartbeatCheckInterval` → **NOT used anywhere in Go code** (it's registered in `mapstructure` but no consumer exists)
- `MinCompatibleVersion` / `VersionCheckEnabled` → used by `EdgeNodeService.SetVersionConfig()`

---

## Timing Analysis (Default Settings)

| Parameter | Value | Description |
|-----------|-------|-------------|
| HeartbeatTimeoutSec | 15s | Node considered offline if no heartbeat for 15s |
| HeartbeatCheckInterval | 10s | Intended check interval (but task NOT scheduled) |
| Earliest offline detection | N/A | Periodic check never runs |
| Only offline path | LWT or manual | MQTT LWT is the only automatic offline path |

---

## What's Missing / Broken

1. **❌ Periodic schedule not registered** — The `edge_node:status_check` handler exists but is never scheduled to run. This is the primary bug. It should be registered in `RegisterPeriodicTasks()` as `"*/10 * * * * *"` (every 10 seconds).

2. **No health check escalation** — No concept of "probationary" offline (e.g., "offline-pending" after 1 timeout, "offline-confirmed" after 2 timeouts).

3. **No engine-side heartbeat configuration** — The Go server's heartbeat config is not communicated to the engine; the engine must independently know the interval.

4. **No heartbeat jitter** — All nodes checked at the same time, causing a thundering herd when many nodes time out simultaneously.

5. **No `last_heartbeat` null handling** — `FindTimedOutNodes` only finds nodes with `last_heartbeat < cutoff`. Nodes with `last_heartbeat IS NULL` (never heartbeated) are not detected.

6. **`heartbeat_check_interval_sec` unused** — The config field exists but has no consumer. The intended check interval is not wired anywhere.
