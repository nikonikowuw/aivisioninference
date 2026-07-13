# EdgeNode Model

## File Path
`/Users/niko/dev/go/aivisioninference/app/internal/model/edge_node.go`

## Key Type
`type EdgeNode struct`

## Status Constants
```go
NodeStatusOnline   = "online"
NodeStatusOffline  = "offline"
NodeStatusError    = "error"
NodeStatusDisabled = "disabled"
```

## Fields

| Field | GORM Type | JSON | Description |
|-------|-----------|------|-------------|
| ID | uuid (BaseModel) | id | Primary key |
| Name | varchar(255) not null | name | Node name |
| Description | varchar(500) | description | Description |
| Endpoint | varchar(255) not null | endpoint | HTTP access address |
| AuthToken | text not null | - (never exposed) | JWT auth token |
| Status | varchar(50) default:'offline' | status | online/offline/error/disabled |
| LastHeartbeat | *time.Time (indexed) | last_heartbeat | Last heartbeat timestamp |
| CPUModel | varchar(255) | cpu_model | CPU model |
| GPUModel | varchar(255) | gpu_model | GPU model |
| HALPlatform | varchar(100) | hal_platform | rkmpp/macos/ascend |
| TotalMemory | int64 | total_memory | Total RAM (bytes) |
| CurrentLoad | int default:0 | current_load | Active task count |
| MaxLoad | int default:1 | max_load | Max concurrent tasks |
| EmbeddingCapacity | int default:1 | embedding_capacity | Embedding concurrency capacity |
| MediaDecodeCapacity | int default:0 | media_decode_capacity | Media decode slot capacity |
| MediaEncodeCapacity | int default:0 | media_encode_capacity | Media encode slot capacity |
| MediaEgressCapacityBPS | int64 default:0 | media_egress_capacity_bps | Media egress bandwidth limit (bit/s) |
| MediaMetricsTTLSeconds | int default:0 | media_metrics_ttl_seconds | Media metrics TTL (s) |
| EngineVersion | varchar(100) | engine_version | Engine version string |
| Uptime | int64 | uptime | Uptime in seconds |
| Enabled | bool default:true | enabled | Whether node is enabled |
| Remark | varchar(1000) | remark | Remarks |
| DeletedAt | gorm.DeletedAt (indexed) | - | Soft delete timestamp |

## Key Observations
1. **No `LastSeen` vs `LastHeartbeat` distinction** — Only `LastHeartbeat` exists, used both for liveness and status display.
2. **No `DisconnectedAt` field** — No way to track when a node went offline, which is essential for recovery SLAs.
3. **No `HeartbeatInterval` field** — Node-specific heartbeat interval not stored (uses global config default 15s).
4. **`Status` is a string enum** — Manual validation only at the service/repo level, not DB-level CHECK constraint.
5. **`AuthToken` never exposed in JSON** — Properly uses `json:"-"`.
6. **No consecutive heartbeat failure count** — No counter to distinguish transient vs persistent offline.
7. **`Remark` used for error messages** — When `HeartbeatRequest.Status == "error"`, the `ErrorMessage` is stored in `remark`.

## Table
```sql
-- Generated from GORM model
CREATE TABLE edge_nodes (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description VARCHAR(500),
    endpoint VARCHAR(255) NOT NULL,
    auth_token TEXT NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'offline',
    last_heartbeat TIMESTAMP WITH TIME ZONE,
    cpu_model VARCHAR(255),
    gpu_model VARCHAR(255),
    hal_platform VARCHAR(100),
    total_memory BIGINT,
    current_load INT DEFAULT 0,
    max_load INT DEFAULT 1,
    embedding_capacity INT DEFAULT 1,
    media_decode_capacity INT NOT NULL DEFAULT 0,
    media_encode_capacity INT NOT NULL DEFAULT 0,
    media_egress_capacity_bps BIGINT NOT NULL DEFAULT 0,
    media_metrics_ttl_seconds INT NOT NULL DEFAULT 0,
    engine_version VARCHAR(100),
    uptime BIGINT,
    enabled BOOLEAN DEFAULT true,
    remark VARCHAR(1000),
    created_at TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TIME ZONE,
    deleted_at TIMESTAMP WITH TIME ZONE,
    -- Unique name constraint: "WHERE deleted_at IS NULL"
);
```

## Related File
`/Users/niko/dev/go/aivisioninference/app/internal/dto/edge_node.go` — Request/response DTOs including `HeartbeatRequest`, `HeartbeatResponse`, `CreateEdgeNodeRequest`, `UpdateEdgeNodeRequest`.
