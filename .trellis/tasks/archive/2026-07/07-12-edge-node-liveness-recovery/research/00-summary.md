# Research Summary — Edge Node Liveness & Task Recovery

## Files Found (absolute paths)

### Models
- `/Users/niko/dev/go/aivisioninference/app/internal/model/edge_node.go` — EdgeNode model
- `/Users/niko/dev/go/aivisioninference/app/internal/model/edge_node_algorithm.go` — EdgeNodeAlgorithm model
- `/Users/niko/dev/go/aivisioninference/app/internal/model/ai_vision_task.go` — AIVisionTask model

### DTOs
- `/Users/niko/dev/go/aivisioninference/app/internal/dto/edge_node.go` — All edge node request/response DTOs

### Handlers (HTTP)
- `/Users/niko/dev/go/aivisioninference/app/internal/handler/edge_node.go` — EdgeNodeHandler (CRUD + Heartbeat + Deploy)
- `/Users/niko/dev/go/aivisioninference/app/internal/middleware/edge_node.go` — EdgeNodeMiddleware (JWT auth for nodes)
- `/Users/niko/dev/go/aivisioninference/app/internal/router/edge_node.go` — Route registration

### Handlers (MQTT)
- `/Users/niko/dev/go/aivisioninference/app/internal/handler/edge_mqtt_handler.go` — EdgeMqttHandler (all edge MQTT topics)
- `/Users/niko/dev/go/aivisioninference/app/internal/server/mqtt_server.go` — MqttServer (topic subscription)
- `/Users/niko/dev/go/aivisioninference/app/internal/pkg/mqttmux/mux.go` — Topic multiplexer

### Services
- `/Users/niko/dev/go/aivisioninference/app/internal/service/edge_node.go` — EdgeNodeService (business logic)
- `/Users/niko/dev/go/aivisioninference/app/internal/service/mqtt_engine_client.go` — MqttEngineClient (MQTT commands)
- `/Users/niko/dev/go/aivisioninference/app/internal/service/ipc_engine_client.go` — EngineClient interface + MockEngineClient
- `/Users/niko/dev/go/aivisioninference/app/internal/service/engine_metrics_store.go` — EngineMetricsStore

### Repositories
- `/Users/niko/dev/go/aivisioninference/app/internal/repository/edge_node.go` — EdgeNodeRepository
- `/Users/niko/dev/go/aivisioninference/app/internal/repository/aivisiontask.go` — AIVisionTaskRepository
- `/Users/niko/dev/go/aivisioninference/app/internal/repository/edge_node_algorithm.go` — EdgeNodeAlgorithmRepository

### Tasks (Asynq)
- `/Users/niko/dev/go/aivisioninference/app/internal/task/edge_node_status.go` — EdgeNodeStatusTask (liveness check)
- `/Users/niko/dev/go/aivisioninference/app/internal/task/edge_node_algorithm_retry.go` — EdgeNodeAlgorithmRetryTask
- `/Users/niko/dev/go/aivisioninference/app/internal/task/edge_worker.go` — EdgeStateWorker (state reconciliation)
- `/Users/niko/dev/go/aivisioninference/app/internal/task/client.go` — Asynq task client wrapper
- `/Users/niko/dev/go/aivisioninference/app/internal/task/server.go` — Asynq server, scheduler setup, periodic registration
- `/Users/niko/dev/go/aivisioninference/app/internal/task/types.go` — Task type constants

### WebSocket
- `/Users/niko/dev/go/aivisioninference/app/internal/pkg/ws/hub.go` — WebSocket Hub

### Configuration
- `/Users/niko/dev/go/aivisioninference/app/internal/config/config.go` — EngineConfig, defaults

### DI / Wire
- `/Users/niko/dev/go/aivisioninference/app/internal/router/wire.go` — Wire provider sets
- `/Users/niko/dev/go/aivisioninference/app/internal/router/deps.go` — Provider functions
- `/Users/niko/dev/go/aivisioninference/app/internal/router/router.go` — Route setup, Asynq mux/scheduler
- `/Users/niko/dev/go/aivisioninference/app/internal/server/deps.go` — Server-level provider functions (MQTT client, etc.)

### Engine (C++)
- `/Users/niko/dev/go/aivisioninference/engine/src/mqtt_control_plane.cpp` — LWT setup, lifecycle publishing, heartbeat

---

## Key Findings Summary

### ✅ Working Correctly

1. **Heartbeat processing** — Full HTTP + MQTT → Service → Repository flow works, with version check, algorithm sync, task recovery, and WebSocket broadcasts.
2. **Task recovery (resume on reconnect)** — When a node comes back online and heartbeats, suspended tasks are automatically resumed.
3. **MQTT LWT** — Engine sets correct LWT with retained messages. Broker delivers "offline" on unexpected disconnect.
4. **State reconciliation** — When heartbeat reports active streams, hash changes trigger reconciliation to kill ghost streams.
5. **Algorithm deployment via heartbeat** — Pending algorithm packages are returned in heartbeat response and status is tracked.

### ❌ Broken / Missing

#### Critical (data loss / stuck state)

1. **`edge_node:status_check` periodic task never runs** — Handler is registered on Asynq mux, but NOT registered as a periodic schedule in `RegisterPeriodicTasks()`. Nodes that disconnect without LWT will stay "online" permanently. Without this, the only automatic offline detection is MQTT LWT.

2. **LWT handler does NOT suspend tasks** — `HandleLifecycle` sets node status to "offline" but leaves running tasks in "running" state. The `suspendNodeTasks` logic exists but is never called from this path. Tasks become orphaned (stuck in "running" on a dead node).

3. **No task suspension from any automated path** — Neither the (non-running) periodic check nor the (task-skip) LWT handler reliably suspends tasks. The only path that CAN suspend tasks (`EdgeNodeStatusTask.suspendNodeTasks`) is unreachable.

4. **Transaction not truly atomic for heartbeat** — `Transaction()` creates a tx, but inner repo method calls use their own `*gorm.DB` from the original `r.db`, not the transactional one. The three heartbeat steps (update fields, sync algorithms, resume tasks) are NOT in the same DB session.

#### High (reliability / correctness)

5. **No concurrent heartbeat protection** — No pessimistic locking or mutex for concurrent HTTP + MQTT heartbeats on the same node.

6. **Recovery only works for properly suspended tasks** — If a task was left in "running" state when node died, recovery flow only looks for "suspended" tasks. Orphaned "running" tasks are never recovered.

7. **No node capacity check on task recovery** — When resuming suspended tasks, the system doesn't verify the node has available capacity (CurrentLoad < MaxLoad).

8. **`heartbeat_check_interval_sec` config unused** — Field exists in config struct but no code reads it. The intended check interval is never used.

9. **No `last_heartbeat` null handling** — `FindTimedOutNodes` only finds nodes with `last_heartbeat < cutoff`. Nodes with NULL `last_heartbeat` (never heartbeated) are missed.

#### Medium (observability / UX)

10. **No DisconnectedAt field** — EdgeNode model can't track when it went offline, making recovery SLAs impossible to measure.

11. **No heartbeat failure counter** — No way to distinguish transient network glitch from permanent node failure.

12. **No "online" lifecycle handling** — Engine publishes "online" on connect, but server ignores it. Only subsequent heartbeat brings node to "online" status.

13. **WebSocket broadcasts are global** — All edge-node-status and task-status events go to ALL connected clients, not filtered by user's node access.

14. **MQTT heartbeat lacks authentication** — No JWT or HMAC for MQTT heartbeats (broker-trusted only).

#### Low (edge cases / future)

15. **No "stop_all" engine command** — Must iterate and stop streams one by one.

16. **No task re-scheduling** — When node goes offline, tasks are just suspended. No mechanism to re-assign to another online node.

17. **No command retry** — MQTT command timeouts are not retried.

---

## PRD Requirements vs Current State

*(To be filled in based on the actual PRD document)*

| Requirement | Status | Gap |
|-------------|--------|-----|
| Node liveness detection | ❌ Broken | Periodic check not scheduled; only LWT works |
| Task suspension on node offline | ❌ Broken | No code path suspends tasks |
| Task recovery on node reconnection | ✅ Working | Heartbeat handler resumes suspended tasks |
| Graceful offline (engine shutdown) | ✅ Working | Engine publishes "offline" lifecycle message |
| Unexpected offline (crash / network) | ⚠️ Partial | LWT fires but tasks not suspended |
| WebSocket notifications | ✅ Working | Status and task events broadcasted |
| Configurable heartbeat timeout | ✅ Working | `heartbeat_timeout_sec` config (default 15s) |
| Configurable check interval | ❌ Broken | `heartbeat_check_interval_sec` config parsed but unused |

---

## Recommended Priority Order (for Design Document)

1. Register `edge_node:status_check` as periodic task — fix the missing cron entry
2. Add task suspension to LWT handler — the `HandleLifecycle` must also call `suspendNodeTasks`
3. Add mutual exclusion for concurrent heartbeat processing — use `FindByIDForUpdate` with pessimistic lock
4. Add recovery for orphaned "running" tasks — reconcile tasks stuck in "running" on node timeout
5. Add capacity check before task recovery — verify CurrentLoad vs MaxLoad
6. Handle NULL `last_heartbeat` in timeout query
7. Add "online" lifecycle handling — react to engine's "online" publication
8. Add `DisconnectedAt` and failure counter fields to EdgeNode model
9. Implement task re-scheduling to alternative nodes
10. Add MQTT message authentication
