# System Metrics Collection & Dashboard — Design

## C++ Engine Extensions

### Module: MetricsFlattener (new)

```
EngineConfig → DeviceMonitor → DeviceSnapshot
                                    ↓
                            MetricsFlattener
                                    ↓
                    FlatMetrics {
                      cpu_usage, cpu_load_1m/5m/15m,
                      memory_usage, memory_used_bytes,
                      memory_total_bytes,
                      disks: [{path, total, used, percent}],
                      net_rx_bytes, net_tx_bytes,
                      net_rx_speed, net_tx_speed,
                      uptime, process_count, thread_count,
                      temperature
                    }
                                    ↓
                    HeartbeatReporter::BuildHeartbeatPayload()
```

File: `engine/src/monitor/metrics_flattener.cpp`
- Gets DeviceSnapshot via shared_ptr (zero-copy read from DeviceMonitor)
- Computes network speed via delta-tracking (store previous RX/TX, divide by elapsed)
- Temperature: reads from DeviceSnapshot if available, else falls back to thermal zone

### HeartbeatReporter Extensions

`BuildHeartbeatPayload()` in `heartbeat_reporter.cpp`:
- Accepts `DeviceMonitor*` in constructor or via setter
- Appends new fields to existing JSON:
```json
{
  ...existing fields...,
  "cpu_load_1m": 0.5,
  "cpu_load_5m": 0.3,
  "cpu_load_15m": 0.2,
  "disks": [{"path": "/", "total": 64000000000, "used": 32000000000, "percent": 50.0}],
  "net_rx_bytes": 1000000000,
  "net_tx_bytes": 500000000,
  "net_rx_speed": 1250000,
  "net_tx_speed": 625000,
  "process_count": 120,
  "thread_count": 450,
  "temperature": 45.5
}
```

### DeviceMonitor Probe Extensions

In `device_monitor.cpp`, extend light/expensive probe cycles:
- Disk usage: read `/proc/mounts` for mount points, then `statfs()` calls
- Network: read `/proc/net/dev` cumulative bytes per interface
- Load average: read `/proc/loadavg`
- Process/thread count: read `/proc/stat` process count, or count `/proc/[0-9]*` entries
- Temperature: read `/sys/class/thermal/thermal_zone*/temp`

## Go Control Plane

### New Model: EdgeNodeMetrics

File: `app/internal/model/edge_node_metrics.go`

```go
type EdgeNodeMetrics struct {
    BaseModel
    NodeID    string    `gorm:"type:char(36);not null;index:idx_node_created_at" json:"node_id"`
    CreatedAt time.Time `gorm:"index;not null" json:"created_at"`

    CPUUsage    float64 `gorm:"type:double precision;not null;default:0" json:"cpu_usage"`
    CPULoad1m   float64 `gorm:"type:double precision;not null;default:0" json:"cpu_load_1m"`
    CPULoad5m   float64 `gorm:"type:double precision;not null;default:0" json:"cpu_load_5m"`
    CPULoad15m  float64 `gorm:"type:double precision;not null;default:0" json:"cpu_load_15m"`

    MemoryUsage  float64 `gorm:"type:double precision;not null;default:0" json:"memory_usage"`
    MemoryUsed   int64   `gorm:"type:bigint;not null;default:0" json:"memory_used"`
    MemoryTotal  int64   `gorm:"type:bigint;not null;default:0" json:"memory_total"`

    DiskUsage    datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'" json:"disk_usage"`

    NetRXBytes   int64   `gorm:"type:bigint;not null;default:0" json:"net_rx_bytes"`
    NetTXBytes   int64   `gorm:"type:bigint;not null;default:0" json:"net_tx_bytes"`
    NetRXSpeed   float64 `gorm:"type:double precision;not null;default:0" json:"net_rx_speed"`
    NetTXSpeed   float64 `gorm:"type:double precision;not null;default:0" json:"net_tx_speed"`

    Uptime          int64   `gorm:"type:bigint;not null;default:0" json:"uptime"`
    ProcessCount    int32   `gorm:"type:int;not null;default:0" json:"process_count"`
    ThreadCount     int32   `gorm:"type:int;not null;default:0" json:"thread_count"`
    Temperature     float64 `gorm:"type:double precision;not null;default:0" json:"temperature"`

    CurrentLoad     int32   `gorm:"type:int;not null;default:0" json:"current_load"`
    EngineVersion   string  `gorm:"type:varchar(100)" json:"engine_version"`
    HALPlatform     string  `gorm:"type:varchar(100)" json:"hal_platform"`
}
```

Use BRIN index on `(node_id, created_at)` for efficient time-range scans.

### New Repository

File: `app/internal/repository/edge_node_metrics.go`

```go
type EdgeNodeMetricsRepository struct { db *gorm.DB }

func (r *EdgeNodeMetricsRepository) Create(ctx, metrics) error
func (r *EdgeNodeMetricsRepository) Query(ctx, nodeID string, opts MetricsQueryOpts) ([]EdgeNodeMetrics, int64, error)

type MetricsQueryOpts struct {
    Metric      string    // "cpu_usage", "memory_usage", etc.
    From        time.Time
    To          time.Time
    Aggregation string    // "avg", "max", "min" (optional)
    Interval    string    // aggregation window like "5m", "1h" (optional)
    Page        int
    PageSize    int
}
```

Query with time-based aggregation:

```sql
SELECT
  date_trunc(@interval, created_at) AS t,
  avg(@metric_name) AS v
FROM edge_node_metrics
WHERE node_id = ? AND created_at BETWEEN ? AND ?
GROUP BY t
ORDER BY t
LIMIT ? OFFSET ?
```

### Extended Heartbeat Handling

In `service/edge_node_service.go` `HandleHeartbeat()`:
- After `nodeRepo.UpdateHeartbeatFields()` succeeds, write `EdgeNodeMetrics` record
- Use `go func()` with context timeout to avoid blocking heartbeat response (metrics write is non-critical)

### New Handler

File: `app/internal/handler/edge_node_metrics.go`

```
GET /api/v1/edge-nodes/{id}/metrics?metric=cpu_usage&from=...&to=...&aggregation=avg&interval=5m&page=1&page_size=500
GET /api/v1/edge-nodes/overview
```

### Metrics Retention Task

File: `app/internal/task/metrics_retention.go`
- Asynq cron task, default daily
- DELETE FROM edge_node_metrics WHERE created_at < NOW() - INTERVAL '7 days'
- Configurable via METRICS_RETENTION_DAYS env

### WS Hub Extension

Broadcast `edge-node-metrics` event on each heartbeat (under load threshold — skip if too frequent):
```json
{
  "type": "edge-node-metrics",
  "payload": {
    "node_id": "...",
    "cpu_usage": 45.2,
    "memory_usage": 62.1,
    "disk_usage_percent": 38.0,
    "uptime": 86400
  }
}
```

## React Frontend

### Overview Page (new)

File: `web/src/views/admin/devices/edge-nodes/overview.tsx`
- 6 stat cards in a grid: total nodes, online, offline, error, alert count, total tasks
- Donut chart for status distribution

### EdgeNodeList Extension

In `index.tsx`:
- Add columns: CPU%, MEM%, Disk%, Uptime
- Values updated via `edge-node-metrics` WS event

### EdgeNodeDetail Extension

In `detail.tsx`:
- Add "Metrics" tab section below basic info
- 4 line charts using Recharts (AreaChart): CPU, Memory, Disk, Network
- Time range button group: 1h / 6h / 24h / 7d
- Fetch data from `/api/v1/edge-nodes/{id}/metrics`
- Auto-refresh on tab focus or WS event

### MetricsTimeSeries Component

File: `web/src/components/charts/MetricsTimeSeries.tsx`
- Reusable line/area chart component
- Props: data points, label, unit, color, time range, loading state
- Responsive: single column on mobile, 2x2 grid on desktop

## Dependencies

- `recharts`: already in project, used for metrics charts
- `@chakra-ui/stat`: stat card component (available)
- No new npm packages required

## Wire Registration

- New `EdgeNodeMetricsRepository` provider in `app/internal/server/deps.go`
- New `EdgeNodeMetricsHandler` + service in `app/internal/router/deps.go` / `wire.go`
- Route registration in existing edge node router group
- Metrics retention cron in `app/internal/task/` with Asynq registration
