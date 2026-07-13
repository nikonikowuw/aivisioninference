# Edge Node Monitoring — Design Document

## Overview

借鉴哪吒监控 (Nezha) 的产品思路，在 AIVisionInference 现有 Edge Node 架构上自建完整的节点监控系统。三个递进阶段：指标采集与可视化 → 告警引擎 → 远程运维。

## Data Flow

```
┌──────────────────────────────────────────────────────────────┐
│ C++ Engine (Edge Node)                                        │
│                                                               │
│  DeviceMonitor (5s/20s intervals)                             │
│   ├─ /proc/stat          → CPU usage                          │
│   ├─ /proc/meminfo       → Memory usage                       │
│   ├─ /proc/diskstats     → Disk I/O + usage per mount         │
│   ├─ /proc/net/dev       → Network RX/TX bytes                │
│   ├─ /proc/loadavg       → Load average                       │
│   ├─ /proc/uptime        → System uptime                      │
│   ├─ /sys/class/thermal/ → Temperature                        │
│   ├─ /sys/class/drm/     → GPU usage (if available)           │
│   └─ /sys/kernel/npu/    → NPU usage (RKNN/Ascend if avail)   │
│                                                               │
│  DeviceSnapshot → MetricsFlattener → HeartbeatPayload         │
│                                                               │
│  HeartbeatReporter (5s interval)                              │
│   └─ HTTP POST /api/v1/edge-nodes/{id}/heartbeat             │
│      └─ Body: { cpu_usage, memory_usage, disk_usage[{path,   │
│                  total, used, percent}], net_rx_bytes,         │
│                  net_tx_bytes, net_rx_speed, net_tx_speed,     │
│                  uptime, temperature, process_count,            │
│                  ... legacy fields }                           │
└───────────────────────┬──────────────────────────────────────┘
                        │
                        ▼
┌──────────────────────────────────────────────────────────────┐
│ Go Control Plane (Server)                                     │
│                                                               │
│  EdgeNodeHandler.Heartbeat()                                  │
│   ├─ Update EdgeNode record (status, load, version...)        │
│   ├─ Persist metrics snapshot to edge_node_metrics table      │
│   ├─ Broadcast via WS Hub to React frontend                   │
│   └─ [Phase 2] Alert Engine: evaluate rules → fire/resolve    │
│                                                               │
│  Metrics Query API (REST)                                     │
│   GET /api/v1/edge-nodes/{id}/metrics?metric=cpu_usage        │
│        &from=2026-07-10T00:00:00Z&to=2026-07-12T23:59:59Z     │
│        &aggregation=avg&interval=5m&page=1&page_size=500      │
│   → Returns time-series data points with pagination           │
│   → Supports aggregation: avg/max/min by configurable window  │
│                                                               │
│  Metrics Retention Cron                                       │
│   DELETE FROM edge_node_metrics WHERE created_at < NOW() - 7d │
│   (configurable via METRICS_RETENTION_DAYS env var)           │
│                                                               │
│  [Phase 2] Alert Engine                                       │
│   ├─ Worker: after each heartbeat ingestion                   │
│   ├─ Load active rules for this node                          │
│   ├─ Evaluate each rule against latest metrics                │
│   ├─ If threshold met for ≥ duration: create/fire event       │
│   ├─ If threshold no longer met & event firing: resolve event │
│   └─ If event fired & silence window not expired: send notify │
│                                                               │
│  [Phase 2] Notifier (Provider Pattern)                        │
│   ├─ WebhookNotifier: POST JSON to configurable URL           │
│   ├─ EmailNotifier: SMTP send via net/smtp                    │
│   ├─ TelegramNotifier: bot API call                           │
│   └─ [extensible] Register new provider via interface          │
│                                                               │
│  [Phase 3] Command Executor Proxy                             │
│   ├─ MQTT command: "shell_exec" → C++ executes → callback     │
│   └─ WebSocket proxy: browser ↔ Engine PTY                    │
│                                                               │
└───────────────────────┬──────────────────────────────────────┘
                        │ WS Broadcast
                        ▼
┌──────────────────────────────────────────────────────────────┐
│ React Admin Panel                                             │
│                                                               │
│  EdgeNodeOverview (Dashboard page)                            │
│   ├─ Stat cards: online/offline/error/total/alert counts      │
│   └─ Status distribution donut chart                          │
│                                                               │
│  EdgeNodeList (list page, enhanced)                           │
│   └─ Additional columns: CPU%, MEM%, Uptime, Status indicator │
│                                                               │
│  EdgeNodeDetail (detail page, enhanced)                       │
│   ├─ Real-time CPU/MEM/Disk/NET line charts (Recharts)        │
│   ├─ Time range selector: 1h/6h/24h/7d                       │
│   └─ Online rate stats                                        │
│                                                               │
│  [Phase 2] AlertRuleList/AlertRuleForm                        │
│   └─ CRUD management for alert rules                          │
│  [Phase 2] AlertHistoryList                                   │
│   └─ List of firing/resolved events with filters              │
│  [Phase 3] ScheduledTaskList/TaskForm/LogViewer               │
│  [Phase 3] WebTerminal (xterm.js component)                   │
│                                                               │
└──────────────────────────────────────────────────────────────┘
```

## Database Schema (New Tables)

### edge_node_metrics

```sql
CREATE TABLE edge_node_metrics (
    id              CHAR(36) PRIMARY KEY,
    node_id         CHAR(36) NOT NULL REFERENCES edge_nodes(id) ON DELETE CASCADE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- CPU
    cpu_usage       DOUBLE PRECISION NOT NULL DEFAULT 0,   -- 0-100
    cpu_load_1m     DOUBLE PRECISION NOT NULL DEFAULT 0,
    cpu_load_5m     DOUBLE PRECISION NOT NULL DEFAULT 0,
    cpu_load_15m    DOUBLE PRECISION NOT NULL DEFAULT 0,

    -- Memory
    memory_usage    DOUBLE PRECISION NOT NULL DEFAULT 0,   -- 0-100
    memory_used     BIGINT NOT NULL DEFAULT 0,             -- bytes
    memory_total    BIGINT NOT NULL DEFAULT 0,             -- bytes

    -- Disk (JSON array aggregated from /proc or per-mount)
    -- stored as JSONB: [{"path":"/","total":64000000000,"used":32000000000,"percent":50.0}]
    disk_usage      JSONB NOT NULL DEFAULT '[]',

    -- Network
    net_rx_bytes    BIGINT NOT NULL DEFAULT 0,
    net_tx_bytes    BIGINT NOT NULL DEFAULT 0,
    net_rx_speed    DOUBLE PRECISION NOT NULL DEFAULT 0,   -- bytes/s
    net_tx_speed    DOUBLE PRECISION NOT NULL DEFAULT 0,   -- bytes/s

    -- System
    uptime          BIGINT NOT NULL DEFAULT 0,             -- seconds
    process_count   INT NOT NULL DEFAULT 0,
    thread_count    INT NOT NULL DEFAULT 0,
    temperature     DOUBLE PRECISION NOT NULL DEFAULT 0,   -- Celsius

    -- Engine-specific (from existing MetricsReporter)
    worker_count        INT NOT NULL DEFAULT 0,
    idle_worker_count   INT NOT NULL DEFAULT 0,
    active_stream_count INT NOT NULL DEFAULT 0,
    decode_sessions     INT NOT NULL DEFAULT 0,
    encode_sessions     INT NOT NULL DEFAULT 0,
    current_load        INT NOT NULL DEFAULT 0,

    INDEX idx_node_created (node_id, created_at DESC)
);
-- Partition hint: use BRIN index on created_at for large volumes
```

### alert_rules (Phase 2)

```sql
CREATE TABLE alert_rules (
    id              CHAR(36) PRIMARY KEY,
    name            VARCHAR(255) NOT NULL,
    node_id         CHAR(36) REFERENCES edge_nodes(id) ON DELETE CASCADE,
    -- NULL node_id = applies to all nodes (global rule)
    metric_type     VARCHAR(50) NOT NULL,
    -- cpu_usage, memory_usage, disk_usage, node_offline, node_error, temperature
    operator        VARCHAR(5) NOT NULL,  -- >, >=, <, <=, ==
    threshold       DOUBLE PRECISION NOT NULL,
    duration_seconds INT NOT NULL DEFAULT 0,  -- how long condition must hold
    silence_minutes  INT NOT NULL DEFAULT 60, -- min between re-notifications
    enabled         BOOLEAN NOT NULL DEFAULT TRUE,
    notify_channels JSONB DEFAULT '[]',  -- ["webhook","email"]
    description     TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### alert_events (Phase 2)

```sql
CREATE TABLE alert_events (
    id              CHAR(36) PRIMARY KEY,
    rule_id         CHAR(36) NOT NULL REFERENCES alert_rules(id),
    node_id         CHAR(36) NOT NULL REFERENCES edge_nodes(id),
    metric_value    DOUBLE PRECISION NOT NULL,
    status          VARCHAR(20) NOT NULL DEFAULT 'firing',
    -- firing / resolved / acknowledged
    fired_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at     TIMESTAMPTZ,
    acknowledged_by VARCHAR(255),
    acknowledged_at TIMESTAMPTZ,
    notify_sent     BOOLEAN NOT NULL DEFAULT FALSE,
    notify_sent_at  TIMESTAMPTZ
);
```

### edge_node_scheduled_tasks (Phase 3)

```sql
CREATE TABLE edge_node_scheduled_tasks (
    id              CHAR(36) PRIMARY KEY,
    node_id         CHAR(36) NOT NULL REFERENCES edge_nodes(id),
    name            VARCHAR(255) NOT NULL,
    command         TEXT NOT NULL,
    cron_expr       VARCHAR(100),  -- NULL = one-shot
    status          VARCHAR(20) NOT NULL DEFAULT 'active',
    timeout_seconds INT NOT NULL DEFAULT 30,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE edge_node_task_executions (
    id              CHAR(36) PRIMARY KEY,
    task_id         CHAR(36) NOT NULL REFERENCES edge_node_scheduled_tasks(id),
    status          VARCHAR(20) NOT NULL,  -- running / success / failed / timeout
    stdout          TEXT,
    stderr          TEXT,
    exit_code       INT,
    started_at      TIMESTAMPTZ,
    finished_at     TIMESTAMPTZ
);
```

## C++ Engine Extensions

### MetricsFlattener (new module)

Transforms `DeviceSnapshot` (structured tree) into the flat JSON fields required by `HeartbeatRequest`:

```
DeviceSnapshot → {
  cpu_usage, cpu_load_1m/5m/15m,
  memory_usage, memory_used, memory_total,
  disk_usage: [{path, total, used, percent}],
  net_rx_bytes, net_tx_bytes, net_rx_speed, net_tx_speed,
  uptime, process_count, thread_count, temperature
}
```

- Reads from DeviceMonitor's cached snapshot (shared_ptr, zero-copy)
- Computes network speed from delta between snapshots
- CPU load from /proc/loadavg (already in DeviceSnapshot)
- Disk usage from /proc/mounts + statfs (new probe method needed)
- Process/thread count from /proc/stat (already available)

### HeartbeatReporter extension

- `BuildHeartbeatPayload()` now includes all fields from MetricsFlattener
- Backward compatible: existing fields (cpu_usage, memory_usage) remain at same positions
- New fields added without removing old ones

### Command Executor (Phase 3, new module)

- MQTT command handler for `shell_exec`: receives command string + timeout
- Runs via `popen()` or `fork+exec` with timeout (SIGALRM or timerfd)
- Captures stdout + stderr as separate buffers
- Reports result via HTTP callback to Go control plane

### PTY Module (Phase 3, new module)

- Opens `/dev/ptmx` for each WebSocket session
- MQTT-based proxy: Go WS → MQTT pty_write → Engine pty → MQTT pty_output → Go WS
- Session timeout handled in Go side (inactivity kill via MQTT)

## Go API Extensions

### Phase 1

```
GET  /api/v1/edge-nodes/{id}/metrics
  ?metric=cpu_usage          (required: metric type name)
  &from=2026-07-10T00:00:00Z (optional, default 24h ago)
  &to=2026-07-12T23:59:59Z   (optional, default now)
  &aggregation=avg            (optional: avg/max/min, default raw)
  &interval=5m               (optional: aggregation window, default raw)
  &page=1                    (optional, default 1)
  &page_size=500             (optional, max 1000)
  → { "list": [{ "t": "ISO8601", "v": 45.2 }], "total": 10000 }

GET  /api/v1/edge-nodes/overview
  → { "total": 10, "online": 7, "offline": 2, "error": 1, "alert_count": 3 }
```

### Phase 2

```
CRUD /api/v1/alert-rules
GET  /api/v1/alert-events?node_id=&rule_id=&status=&from=&to=
POST /api/v1/alert-events/{id}/acknowledge
```

### Phase 3

```
CRUD /api/v1/edge-nodes/{id}/scheduled-tasks
GET  /api/v1/edge-nodes/{id}/task-executions
WS   /api/v1/edge-nodes/{id}/terminal  → WebSocket upgrade to PTY proxy
```

## Key Design Decisions

1. **No new connection from edge**: all metrics flow through existing heartbeat HTTP POST. Zero extra port/connection overhead.
2. **Flat heartbeat JSON**: avoid nesting that would break existing parsing. New fields are optional at the end of the request body.
3. **PostgreSQL for timeseries**: start with plain PG + BRIN index on `(node_id, created_at)`. If data volume exceeds ~100M rows, migrate to TimescaleDB hypertable under same schema.
4. **Alert engine synchronous to heartbeat**: rule evaluation happens inline in HandleHeartbeat after metrics persist. For Phase 2, evaluation is O(N_rules × N_nodes) — acceptable at current scale (<1000 nodes). Async job queue if needed later.
5. **WebSocket for real-time**: existing WS Hub already pushes `edge-node-status` events. Metrics data will be pushed via same hub (new `edge-node-metrics` event type) for live frontend updates.
6. **Command execution isolation**: each shell command runs in its own process with resource limits (timeout, stdbuf). No persistent shell — each exec is `fork+exec` or `popen()` with strict timeout.
7. **Web Terminal via MQTT tunnel**: Go WS receives PTY output from edge via MQTT and relays to browser WS; browser input goes Go WS → MQTT → Engine PTY. No direct browser→edge connections needed.

## File Change Map

### C++ Engine (Phase 1)
- NEW `engine/src/monitor/metrics_flattener.cpp`
- NEW `engine/include/monitor/metrics_flattener.h`
- MOD `engine/src/monitor/heartbeat_reporter.cpp` — expand payload building
- MOD `engine/include/monitor/heartbeat_reporter.h`
- MOD `engine/CMakeLists.txt` — add new sources
- MOD `engine/src/monitor/device_monitor.cpp` — add disk/net/temp/process probes

### Go Control Plane (Phase 1)
- NEW `app/internal/model/edge_node_metrics.go`
- NEW `app/internal/repository/edge_node_metrics.go`
- NEW `app/internal/service/edge_node_metrics.go`
- NEW `app/internal/handler/edge_node_metrics.go`
- MOD `app/internal/service/edge_node.go` — extend HandleHeartbeat to persist metrics
- MOD `app/internal/dto/edge_node.go` — extend HeartbeatRequest with new fields
- MOD `app/internal/router/edge_node.go` — add metrics routes
- MOD `app/cmd/migrate/main.go` or new migration file for new tables
- NEW `app/internal/task/metrics_retention.go` — Asynq cron for cleanup

### React Frontend (Phase 1)
- NEW `web/src/views/admin/devices/edge-nodes/overview.tsx` — dashboard page
- MOD `web/src/views/admin/devices/edge-nodes/index.tsx` — add real-time columns
- MOD `web/src/views/admin/devices/edge-nodes/detail.tsx` — add charts + metrics tabs
- NEW `web/src/components/charts/MetricsTimeSeries.tsx` — reusable line chart component
- NEW `web/src/services/edgeNodeMetrics.ts` — API client
- MOD i18n files across all 6 languages

### Phase 2 & 3 file maps will be detailed in child tasks' design.md
