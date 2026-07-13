# Heartbeat Handlers — Full Flow

## Overview
There are two heartbeat entry points:
1. **HTTP** — `POST /api/v1/edge-nodes/{id}/heartbeat` (node-specific JWT auth)
2. **MQTT** — Topic `aivision/edge/{node_id}/status/heartbeat` (no auth, broker-trusted)

Both ultimately call the same service method.

---

## 1. HTTP Heartbeat Handler

### Handler
- **File:** `/Users/niko/dev/go/aivisioninference/app/internal/handler/edge_node.go`
- **Function:** `EdgeNodeHandler.Heartbeat(c *gin.Context)`
- **Route:** `v1.POST("/edge-nodes/:id/heartbeat", nodeMiddleware.AuthNode(), nodeHandler.Heartbeat)`
- **Route File:** `/Users/niko/dev/go/aivisioninference/app/internal/router/edge_node.go`

```go
func (h *EdgeNodeHandler) Heartbeat(c *gin.Context) {
    id := c.Param("id")
    var req dto.HeartbeatRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        response.Err(c, badRequestError(c, err))
        return
    }
    res, err := h.svc.HandleHeartbeat(c.Request.Context(), id, &req)
    if err != nil {
        response.Err(c, err)
        return
    }
    response.OK(c, res)
}
```

- Authenticated via `EdgeNodeMiddleware.AuthNode()` which validates JWT and checks node ID match.

### Middleware
- **File:** `/Users/niko/dev/go/aivisioninference/app/internal/middleware/edge_node.go`
- **Type:** `EdgeNodeMiddleware`
- Validates `Authorization: Bearer <token>` header
- Parses node JWT token via `jwtManager.ParseNodeToken()`
- Ensures path `:id` matches token's `NodeID` claim

---

## 2. MQTT Heartbeat Handler

### Handler
- **File:** `/Users/niko/dev/go/aivisioninference/app/internal/handler/edge_mqtt_handler.go`
- **Function:** `EdgeMqttHandler.HandleHeartbeat(msg mqtt.Message)`
- **Topic Registration:** `aivision/edge/+/status/heartbeat`
- **Registered in:** `/Users/niko/dev/go/aivisioninference/app/internal/server/mqtt_server.go`
  ```go
  s.mux.Register("aivision/edge/+/status/heartbeat", s.handler.HandleHeartbeat)
  ```

```go
func (h *EdgeMqttHandler) HandleHeartbeat(msg mqtt.Message) {
    topicParts := strings.Split(msg.Topic(), "/")
    // topicParts[2] is nodeID
    nodeID := topicParts[2]

    var req dto.HeartbeatRequest
    json.Unmarshal(msg.Payload(), &req)

    // Process heartbeat in the EdgeNodeService
    h.nodeSvc.HandleHeartbeat(context.Background(), nodeID, &req)

    // State reconciliation detection
    sort.Strings(req.ActiveStreams)
    // ... MD5 hash of active streams ...
    // If hash changed from Redis stored value:
    //   1. Store new hash in Redis (5min TTL)
    //   2. Store actual streams in Redis (1hr TTL)
    //   3. Enqueue TaskReconcileEdgeState
}
```

**Key observations:**
- The MQTT handler **additionally** detects active-stream state changes and enqueues reconciliation.
- The HTTP handler does NOT do state reconciliation — this is MQTT-only.
- `HandleHeartbeat` in MQTT handler runs synchronously (blocks the MQTT dispatch goroutine).

---

## 3. Service Layer — `HandleHeartbeat`

### File
`/Users/niko/dev/go/aivisioninference/app/internal/service/edge_node.go`

### Function
```go
func (s *EdgeNodeService) HandleHeartbeat(ctx context.Context, id string, req *dto.HeartbeatRequest) (*dto.HeartbeatResponse, error)
```

### Flow (inside a DB transaction):

1. **Find node by ID** — `s.nodeRepo.FindByID(ctx, id)` — returns error if not found
2. **Check version compatibility** (if enabled) — compares `req.EngineVersion` against `minCompatibleVersion`
3. **Determine new status:**
   - If `req.Status == "error"` → status = `"error"`, remark = `req.ErrorMessage`
   - If node was `"disabled"` → stays `"disabled"`
   - Otherwise → status = `"online"`
4. **Build heartbeat fields map:**
   ```go
   hbFields := map[string]interface{}{
       "last_heartbeat": &now,
       "uptime":         req.Uptime,
       "current_load":   req.CurrentLoad,
       "engine_version": req.EngineVersion,
       "hal_platform":   req.HALPlatform,
       "cpu_model":      req.HardwareInfo.CPUModel,
       "gpu_model":      req.HardwareInfo.GPUModel,
       "total_memory":   req.HardwareInfo.TotalMemory,
       "status":         status,
       "remark":         remark,
   }
   ```
5. **DB Transaction (3 steps):**
   - `UpdateHeartbeatFields()` — updates only allowed columns (status, last_heartbeat, current_load, uptime, engine_version, hal_platform, cpu_model, gpu_model, total_memory, embedding_capacity, remark)
   - `SyncInstalled()` — reconciles installed algorithms list from heartbeat
   - **If status == "online":** find all suspended tasks for this node and `ClearErrorReason()` each → restores to `"running"`
6. **WebSocket broadcast** (after successful transaction):
   - `edge-node-status` event with node_id, status, current_load, engine_version, last_heartbeat
   - `task-status` event for each resumed task
7. **Check pending deployments:**
   - Query `ListPendingByNode()` for pending algorithm deployments
   - Generate presigned URLs for each
   - Update status to `"downloading"`
   - Return `HeartbeatResponse{PendingDeployments: deployments}`

---

## 4. Repository Layer

### File
`/Users/niko/dev/go/aivisioninference/app/internal/repository/edge_node.go`

### Key Methods

| Method | Description | Transaction-safe |
|--------|-------------|-----------------|
| `UpdateHeartbeatFields(id, fields)` | Updates only allowed heartbeat columns | No (but called inside Transaction) |
| `Transaction(ctx, fn)` | Wraps operations in DB transaction | Yes |
| `FindByID(id)` | Find by primary key | No |
| `Update(item)` | Full Save (all fields) | No |
| `FindTimedOutNodes(cutoff)` | Online nodes with heartbeat < cutoff | No |
| `UpdateStatusBatch(ids, status)` | Bulk status update | No |

### Allowed heartbeat columns (whitelist):
```go
allowed := []string{"status", "last_heartbeat", "current_load", "uptime", "engine_version",
    "hal_platform", "cpu_model", "gpu_model", "total_memory", "embedding_capacity", "remark"}
```

### Transaction wrapper:
```go
func (r *EdgeNodeRepository) Transaction(ctx context.Context, fn func(context.Context) error) error {
    return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
        return fn(ctx)
    })
}
```
**Note:** The transaction does NOT propagate the `tx` to inner repository calls — they use their own `db`, not the transactional one. This means the 3 steps in `HandleHeartbeat` are NOT truly atomic at the DB session level. The `Transaction()` only ensures they run within the same database transaction but each inner call creates its own `*gorm.DB` context.

---

## HeartbeatRequest DTO

### File
`/Users/niko/dev/go/aivisioninference/app/internal/dto/edge_node.go`

```go
type HeartbeatRequest struct {
    Uptime              int64                    `json:"uptime" binding:"min=0"`
    CurrentLoad         int                      `json:"current_load" binding:"min=0"`
    CPUUsage            float64                  `json:"cpu_usage" binding:"min=0,max=100"`
    MemoryUsage         float64                  `json:"memory_usage" binding:"min=0,max=100"`
    EngineVersion       string                   `json:"engine_version" binding:"required"`
    HALPlatform         string                   `json:"hal_platform" binding:"required"`
    HardwareInfo        HardwareInfo             `json:"hardware_info" binding:"required"`
    InstalledAlgorithms []InstalledAlgorithmInfo `json:"installed_algorithms"`
    ActiveStreams       []string                 `json:"active_streams"`
    Status              string                   `json:"status" binding:"omitempty"`
    ErrorMessage        string                   `json:"error_message" binding:"omitempty"`
}
```

## What's Missing / Broken

1. **MQTT heartbeat lacks node authentication** — MQTT uses broker-trusted connection, any node could impersonate another if broker is compromised.
2. **Transaction not truly atomic** — `Transaction()` creates a tx but inner repo methods create their own `*gorm.DB` from the original `r.db`, not from the tx. This means the three steps (update heartbeat, sync algorithms, resume tasks) are NOT in the same DB session. They are in the same database transaction due to `gorm.DB.Transaction()`, but the repository methods lose the transactional context.
3. **No mutual exclusion** — If HTTP and MQTT heartbeats arrive concurrently for the same node, there's no locking, leading to lost updates.
4. **No version check for MQTT path** — Version compatibility is checked inside `HandleHeartbeat` regardless of entry point, which is correct.
