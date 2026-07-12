# Edge Node Liveness And Task Recovery

## 1. Scope / Trigger

Use this contract when changing edge-node heartbeat handling, MQTT LWT processing, timeout checks, task suspension/recovery, Asynq registration, or their WebSocket notifications. The control plane must keep the persisted task state, Engine pipeline state, and client-visible state consistent under duplicate events and concurrent operator actions.

## 2. Signatures

The shared offline transition is service-owned and transaction-aware:

```go
func HandleNodeOffline(
    ctx context.Context,
    nodeRepo *repository.EdgeNodeRepository,
    taskRepo *repository.AIVisionTaskRepository,
    hub *ws.Hub,
    node model.EdgeNode,
    suspendedReason string,
    reasonFmt string,
    cutoff *time.Time,
    reasonArgs ...interface{},
) (transitioned bool, err error)
```

Recovery persistence must use conditional repository methods:

```go
ClearNodeOfflineSuspended(ctx context.Context, taskID string) (cleared bool, err error)
UpdateNodeOfflineSuspendedError(ctx context.Context, taskID, errorMsg string) error
```

The Asynq mux must receive the application-owned Hub:

```go
NewAsynqMux(..., scheduler *asynq.Scheduler, hub *ws.Hub) *asynq.ServeMux
```

## 3. Contracts

- `ai_vision_tasks.suspended_reason` is nullable. Supported machine values are `node_offline`, `manual`, `schedule`, and `error`; `NULL` is legacy/unknown and is never auto-recovered.
- HTTP timeout and MQTT LWT must call `HandleNodeOffline`. The node status change and running-task suspension occur in one database transaction.
- Timeout processing uses `MarkOfflineIfTimedOut(nodeID, cutoff)` so a heartbeat newer than the worker snapshot wins. LWT uses the unconditional online-to-offline transition. Duplicate offline events return `transitioned=false` and emit no duplicate WebSocket events.
- WebSocket messages are emitted only after commit and through the same `*ws.Hub` that the server runs for connected clients. Background workers must not create a private Hub.
- Heartbeats update liveness and Engine status fields but never overwrite the administrator-owned `edge_nodes.remark` with Engine error text.
- Only tasks still matching `status=suspended AND suspended_reason=node_offline` may transition to `running` after recovery.
- Recovery calls `StreamManager.RestoreInferenceOnNode`, which requires a successful Engine `StartStream` response before persistence changes. If Engine start fails, the task remains suspended and receives an error reason.
- If the conditional database transition fails or affects zero rows after Engine start, release the restored `infer:<task_id>` consumer to compensate for a concurrent manual state change or persistence failure.
- Register `edge_node:status_check` from `heartbeat_check_interval_sec`; handle and log `Scheduler.Register` errors explicitly.

## 4. Validation & Error Matrix

| Condition | Required result |
| --- | --- |
| `heartbeat_check_interval_sec <= 0` | Skip registration and log a warning |
| Scheduler registration fails | Log an error; do not claim registration succeeded |
| Timeout candidate receives a newer heartbeat | Conditional offline update affects zero rows; leave node/tasks unchanged |
| Duplicate timeout or LWT event | Return `transitioned=false`; no duplicate task or WS transition |
| Task reason is `manual`, `schedule`, `error`, or `NULL` | Do not send an automatic restore command |
| Engine restore fails or times out | Keep `suspended`; preserve `node_offline`; update `error_reason` conditionally |
| Task changes while Engine restore is in flight | Conditional clear affects zero rows; roll back the restored StreamManager consumer |
| DB clear fails after Engine ACK | Roll back the restored StreamManager consumer and keep persisted state unchanged |
| Heartbeat reports `status=error` | Set node status to error, log the Engine error, preserve `remark` |

## 5. Good / Base / Bad Cases

- Good: a timed-out online node is atomically marked offline, its running tasks become `node_offline` suspended, and the shared Hub broadcasts the committed transitions once.
- Base: a repeated LWT arrives after the timeout worker already committed. The conditional node update is a no-op and no duplicate notifications are sent.
- Bad: a worker constructs `ws.NewHub()` locally. Database state changes correctly, but connected administration clients never receive the broadcast.
- Bad: recovery clears suspension by task ID only. A concurrent manual pause can be overwritten after a delayed Engine ACK.

## 6. Tests Required

- Timeout test: assert fresh heartbeats are not marked offline and stale nodes are.
- Idempotency test: call `HandleNodeOffline` twice and assert only the first call transitions the node and task.
- Recovery success test: assert the Engine request uses the target node, device, algorithm runtime path, and task ID before the task becomes `running`.
- Recovery filtering test: assert a `manual` suspended task remains suspended after heartbeat recovery.
- Recovery failure test: assert a missing device, Engine error, or timeout keeps the task `node_offline` suspended and records `error_reason`.
- StreamManager rollback test: assert failed `StartStream` leaves no `infer:<task_id>` consumer and a zero reference count.
- Heartbeat ownership test: assert normal and error heartbeats preserve the existing `remark`.
- Wiring/build check: run `cd app && make wire` and `cd app && make build` after changing Hub or scheduler dependencies.

## 7. Wrong vs Correct

### Wrong

```go
hub := ws.NewHub()
task := NewEdgeNodeStatusTask(nodeRepo, taskRepo, hub, timeout)

taskRepo.Model(&model.AIVisionTask{}).
    Where("id = ?", taskID).
    Update("status", model.TaskStatusRunning)
```

The private Hub has no connected clients, and the unconditional update can overwrite a concurrent manual pause.

### Correct

```go
task := NewEdgeNodeStatusTask(nodeRepo, taskRepo, appHub, timeout)

result := db.Model(&model.AIVisionTask{}).
    Where("id = ? AND status = ? AND suspended_reason = ?",
        taskID, model.TaskStatusSuspended, model.SuspendedReasonNodeOffline).
    Updates(map[string]interface{}{
        "status": model.TaskStatusRunning,
        "suspended_reason": nil,
        "error_reason": "",
    })
```

Use `RowsAffected` to decide whether to broadcast `running`; compensate the Engine/StreamManager restore when no row was cleared.
