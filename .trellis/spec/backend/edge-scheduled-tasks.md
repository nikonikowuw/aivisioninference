# Edge Scheduled Tasks — Spec & Contracts

## 1. Scope / Trigger

Use this contract when modifying the edge scheduled task system (`edge_scheduled_tasks`), EdgeNode tags (`edge_node_tags`), the Asynq patrol worker, or the MQTT command dispatch path. The system periodically dispatches MQTT commands to edge nodes based on cron expressions.

## 2. Data Model

### EdgeNodeTag

```go
type EdgeNodeTag struct {
    BaseModel
    Name        string     // unique index, max 100 chars
    Color       string     // hex color, default #1890ff
    Description string     // max 500 chars
    EdgeNodes   []EdgeNode // M:N through edge_node_tag_relations
}
```

**Join table** `edge_node_tag_relations`:
- `edge_node_id` (FK → edge_nodes.id, CASCADE)
- `edge_node_tag_id` (FK → edge_node_tags.id, CASCADE)
- Composite PK on (edge_node_id, edge_node_tag_id)

### EdgeScheduledTask

```go
type EdgeScheduledTask struct {
    BaseModel
    Name             string         // max 255, not null
    Description      string         // max 500
    CronExpr         string         // standard 5-field cron, not null
    TargetType       string         // "single_node" | "tag"
    TargetID         string         // UUID: node_id or tag_id
    CommandName      string         // one of the 9 supported commands
    CommandParams    datatypes.JSON // arbitrary params passed to Engine
    WaitResponse     bool           // whether to wait for MQTT response
    WaitTimeoutSec   int            // default 30
    Enabled          bool           // default true
    MaxRetries       int            // default 3
    RetryIntervalSec int            // default 60
    LastRunAt        *time.Time
}
```

### EdgeScheduledTaskRecord

```go
type EdgeScheduledTaskRecord struct {
    BaseModel
    TaskID          string         // FK reference
    NodeID          string         // target node
    Status          string         // pending | dispatched | success | failed
    TraceID         string         // MQTT trace_id for response matching
    RequestPayload  datatypes.JSON // the serialized request sent
    ResponsePayload datatypes.JSON // response data (sync mode only)
    ErrorMessage    string         // error description
    RetryCount      int            // incremented on retry
    ExecutedAt      *time.Time
    CompletedAt     *time.Time
}
```

## 3. Contracts

### Command Names (Go Constants)

```go
ScheduledTaskCmdStartStream    = "start_stream"
ScheduledTaskCmdStopStream     = "stop_stream"
ScheduledTaskCmdStartPlayback  = "start_playback"
ScheduledTaskCmdStopPlayback   = "stop_playback"
ScheduledTaskCmdStreamStatus   = "stream_status"
ScheduledTaskCmdSelfCheck      = "self_check"
ScheduledTaskCmdFaceLibrary    = "face_library"
ScheduledTaskCmdAlgoWarmup     = "algo_warmup"
ScheduledTaskCmdFaceEmbedding  = "face_embedding"
```

### MQTT Command Dispatch

**Topic pattern**: `aivision/edge/{nodeID}/cmd/{commandName}`

**Fire-and-forget mode** (`wait_response=false`, default):
- Publish JSON payload with `trace_id` + merged `CommandParams`
- Record status = `dispatched`
- No response tracking

**Sync mode** (`wait_response=true`):
- Same publish, then wait on `MqttSyncManager.Wait(ctx, traceID, timeout)`
- Record status = `success` or `failed`
- Response payload saved in `ResponsePayload`

### Asynq Task Registration

```go
// Patrol: every 60 seconds
scheduler.Register("@every 60s", asynq.NewTask(TypeEdgeScheduledTaskPatrol, nil))
// Cleanup: daily at 3:00 AM
scheduler.Register("0 3 * * *", asynq.NewTask(TypeEdgeScheduledTaskCleanup, nil))
```

### Cron Matching Strategy

```go
schedule, _ := cron.ParseStandard(task.CronExpr)
nextTime := schedule.Next(now.Add(-2 * time.Minute))
if !nextTime.IsZero() && !nextTime.After(now) {
    // current time matches the cron expression — execute
}
```

This checks if the cron expression would have fired in the past 2 minutes, preventing both missed executions and duplicate executions within the same patrol cycle.

### Route Registration

```
GET    /edge-scheduled-tasks          — list
POST   /edge-scheduled-tasks          — create
GET    /edge-scheduled-tasks/:id      — get by ID
PUT    /edge-scheduled-tasks/:id      — update
DELETE /edge-scheduled-tasks/:id      — delete
PUT    /edge-scheduled-tasks/:id/toggle — enable/disable
GET    /edge-scheduled-tasks/records   — list records (with task_id/status filter)
POST   /edge-scheduled-tasks/:id/records/:record_id/retry — retry failed record
```

### WebSocket Events

```
scheduled_task.executed  — command dispatched to a node
scheduled_task.completed — command completed (success/failure)
```

## 4. Validation & Error Matrix

| Condition | System Action |
|-----------|---------------|
| Invalid command name | Return `REQ_BAD_REQUEST` at create/update |
| Invalid cron expression | Skip task in patrol, log warning |
| Target type `single_node` with non-existent node | Record fails with error_message |
| Target type `tag` resolves to 0 nodes | Patrol skips, no records created |
| MQTT client is nil | Record set to `failed` with error |
| Sync mode response timeout | Record set to `failed` with timeout message |
| Retry on non-failed record | Return `REQ_BAD_REQUEST` |
| Duplicate tag name | DB unique constraint error |

## 5. Good / Base / Bad Cases

**Good**: A cron expression `0 */2 * * *` fires every 2 hours. The patrol worker detects the match, resolves the target tag to 3 online nodes, creates 3 records, dispatches MQTT commands to all 3, records `dispatched` for each.

**Base**: A tag resolves to 0 nodes (tag has no members). The patrol skips the task silently, logs a warning at debug level. No records are created.

**Bad**: The patrol worker creates a record but the MQTT publish fails because the client is nil. The record transitions `pending → failed` with the error captured.

**Bad**: A record is retried but the task has been soft-deleted since the original execution. `FindByID` returns `ErrNotFound` and the retry fails with `NOT_FOUND`.

## 6. Tests Required

- EdgeScheduledTaskRepository CRUD tests (create, list, find by ID, update, delete)
- EdgeScheduledTaskService tests for command name validation
- EdgeScheduledTaskService tests for target resolution (single_node vs tag)
- EdgeScheduledTaskService RetryRecord tests (rejects non-failed records)
- Patrol cron matching tests (cron matches within window, cron outside window)
- Wire DI test: `make wire` + `make build`

## 7. Wrong vs Correct

### Wrong

```go
// Comparing with time.Now() exactly — misses if patrol is slightly late
next := schedule.Next(time.Now())
if !next.IsZero() && !next.After(time.Now()) {
    executeTask(task)
}
```

The patrol runs every 60 seconds; an exact comparison can miss executions or cause duplicates.

### Correct

```go
// Use a 2-minute lookback window
now := time.Now()
nextTime := schedule.Next(now.Add(-2 * time.Minute))
if !nextTime.IsZero() && !nextTime.After(now) {
    executeTask(task)
}
```

The 2-minute window accounts for the 60-second patrol interval plus scheduling jitter.
