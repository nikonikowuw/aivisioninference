# Protocols And Algorithms

## Control And Event Channels

The current runtime channel is not a pure FlatBuffers IPC system:

- Go publishes MQTT JSON commands through `app/internal/service/mqtt_engine_client.go`.
- C++ subscribes and dispatches commands in `engine/src/mqtt_control_plane.cpp` and `engine/src/command_dispatcher.cpp`.
- C++ builds command responses through `ResponseRouter` and publishes MQTT JSON responses.
- Inference events are MQTT messages whose payload is wrapped by FlatBuffers `ControlEnvelope`.
- Edge-node heartbeat is HTTP: `POST /api/v1/edge-nodes/{id}/heartbeat`.

`proto/flatbuf/README.md` is the source of truth for schema role and generation commands.

## FlatBuffers Schema Evolution

Schema files live in `proto/flatbuf/`:

- `common.fbs`
- `envelope.fbs`
- `commands.fbs`
- `results.fbs`
- `CHANGELOG.md`
- `COMPATIBILITY.md`

When changing schemas:

1. Add optional fields where possible.
2. Do not reorder existing fields.
3. Mark old fields deprecated instead of deleting them.
4. Update `CHANGELOG.md` and `COMPATIBILITY.md`.
5. Regenerate Go output under `app/internal/pkg/controlproto/fbs`.
6. Regenerate C++ output under `engine/include/proto/flatbuf`.
7. Build/test both Go and engine sides.

Do not manually edit generated FlatBuffers code.

## MQTT Command Compatibility

MQTT JSON is the command-control path, so schema changes alone are not enough for command behavior. Update the Go command builder, C++ dispatcher, response router, tests, and frontend/service expectations when command fields change.

Preserve traceability fields such as sequence id, timestamp, node id, task id, and stream id. They are needed to debug cross-process issues.

## Scenario: Explicit AI Stream Node Routing

### 1. Scope / Trigger

- Trigger: changing `EngineClient`, AI task Start/Stop/Restart, `StreamManager`, MQTT stream commands, or edge-state reconciliation.
- This contract separates command destination (`node_id`) from the Engine Pipeline resource (`device_id`) and business task (`task_id`).

### 2. Signatures

```go
type StreamStartRequest struct {
    NodeID   string
    TaskID   string
    DeviceID string
    // stream URL, playback, inference, and algorithm fields omitted
}

StartStream(ctx context.Context, req StreamStartRequest) (StreamInfo, error)
StopStream(ctx context.Context, nodeID, deviceID string) error
StartPlayback(ctx context.Context, req StreamStartRequest) (string, error)
StopPlayback(ctx context.Context, nodeID, deviceID string) error
GetStreamStatus(ctx context.Context, nodeID, deviceID string) (StreamStatus, error)
```

### 3. Contracts

- MQTT topic is `aivision/edge/{node_id}/cmd/{command}`. Never substitute a device/channel ID for `{node_id}`.
- MQTT JSON keeps `device_id` and `trace_id`; start commands also keep `task_id`. AI starts use the persisted task ID, while non-task streams may fall back to `device_id` for compatibility.
- Engine subscribes with its configured node ID and continues to create, query, and destroy Pipelines by payload `device_id`.
- `StreamManager` state is keyed by `(node_id, device_id)`. AI consumers use `infer:{task_id}` and retry from the saved route and start request.
- AI Create, Update, and Start share one admission policy: online, enabled, `max_load > 0`, `current_load < max_load`, and algorithm deployment status `installed`.

### 4. Validation & Error Matrix

| Condition | Result |
|-----------|--------|
| Node missing | `ErrEdgeNodeNotFound` |
| Node offline/error | `ErrEdgeNodeOffline` |
| Node disabled or `enabled=false` | `ErrEdgeNodeDisabled` |
| `max_load <= 0` or `current_load >= max_load` | `ErrEdgeNodeFull` |
| Algorithm deployment missing or not `installed` | `ErrEdgeNodeAlgorithmUnavailable` |
| MQTT stream command has empty node ID | Reject before publish |

### 5. Good/Base/Bad Cases

- Good: task `task-1`, device `camera-1`, node `edge-a` publishes to `aivision/edge/edge-a/cmd/start_stream` with both IDs in the payload.
- Base: two nodes may host `camera-1`; their StreamManager states and consumers remain independent.
- Bad: publishing to `aivision/edge/camera-1/cmd/start_stream`, using a fixed `infer` consumer, or selecting a full node.

### 6. Tests Required

- MQTT topic tests cover Start/Stop/Playback/Status and assert the node ID occupies the topic segment.
- Payload tests assert `device_id`, real AI `task_id`, and non-empty `trace_id`.
- StreamManager tests assert the same device ID is isolated across two nodes and releasing one route does not affect the other.
- Admission tests cover every error-matrix row and stable recommendation ordering by load ratio then node ID.
- Engine tests/build confirm node-specific subscription and device-ID Pipeline operations remain compatible.

### 7. Wrong vs Correct

Wrong:

```go
topic := fmt.Sprintf("aivision/edge/%s/cmd/start_stream", req.DeviceID)
streamManager.Acquire(ctx, task.DeviceChannelID, "infer", metadata)
```

Correct:

```go
topic := edgeCommandTopic(req.NodeID, "start_stream")
streamManager.AcquireOnNode(ctx, StreamRoute{
    NodeID: task.TargetNodeID, DeviceID: task.DeviceChannelID,
}, "infer:"+task.ID, metadata)
```

## Scenario: Preview Capacity Admission

### 1. Scope / Trigger

- Trigger: changing preview admission, Engine playback lifecycle, `EngineMetricsMsg`, or media-capacity reservations.
- The contract spans Engine environment configuration, `PipelineManager`, FlatBuffers, Go metrics caching, and database-backed admission.

### 2. Signatures

- Environment: `NIKO_ENGINE_MAX_PREVIEW_STREAMS=<positive uint32>`.
- FlatBuffers: `EngineMetricsMsg.preview_capacity:uint`, `preview_in_use:uint`, `preview_capacity_valid:bool`, plus `timestamp_ns:uint64`.
- Go admission: `MediaCapacityScheduler.Reserve(ctx, streamKey, lease)` reserves exactly one preview slot.
- Database: `media_capacity_reservations.preview_slots` is `1` for every new preview reservation; existing decode/encode/egress columns are compatibility-only and must not affect admission.

### 3. Contracts

- `preview_capacity` is a device-qualified deployment value owned by Engine configuration. Go must not derive or override it from decoder, encoder, bandwidth, NPU, or `current_load` values.
- `preview_in_use` counts unique pipelines whose playback output is enabled. Pure inference counts zero; inference plus playback counts one.
- `CreatePipeline`, `EnablePlayback`, `DisablePlayback`, and `DestroyPipeline` must keep playback state idempotent so repeated commands cannot change the count twice.
- Go admits only when the node is online/enabled, the snapshot and receive time are within TTL, the validity bit is true, and `preview_in_use + pending + 1 <= preview_capacity`.
- Eligible nodes sort by post-admission ratio `(preview_in_use + pending + 1) / preview_capacity`, then ascending node ID.

### 4. Validation & Error Matrix

| Condition | Result |
|-----------|--------|
| Environment missing, zero, malformed, trailing text, or greater than `uint32` | Log a configuration error, report `preview_capacity_valid=false`, reject new automatic preview admission |
| Old Engine omits appended fields | FlatBuffers zero defaults produce `preview_capacity_valid=false`; reject admission |
| `preview_in_use > preview_capacity` | Treat snapshot as invalid; reject admission |
| Snapshot or Go receive time exceeds TTL | Reject with `stale_metrics` |
| `preview_in_use + pending + 1 > preview_capacity` | Reject with `preview_full` |
| Repeated reservation for the same `stream_key` | Return the existing effective reservation without consuming another slot |

### 5. Good/Base/Bad Cases

- Good: capacity `8`, in use `5`, pending `1` admits one preview and scores `7/8`.
- Base: capacity `1`, in use `0`, pending `0` admits exactly one concurrent requester.
- Bad: validity false, capacity zero, stale timestamp, or total usage equal to capacity rejects the request.

### 6. Tests Required

- Protocol conversion asserts all three preview fields survive direct and enveloped FlatBuffers payloads.
- Engine tests/build assert configured capacity is reported and playback-enabled pipelines determine `preview_in_use`.
- Scheduler tests assert lowest post-admission ratio, node-ID tie break, `preview_full`, stale/invalid metrics, lease expiry, and no oversubscription of the final slot.
- Rolling-upgrade tests/assertions treat a payload without the appended fields as unschedulable.

### 7. Wrong vs Correct

Wrong:

```text
admit = decode_slots && encode_slots && egress_bandwidth
```

Correct:

```text
admit = preview_capacity_valid && fresh && preview_in_use + pending + 1 <= preview_capacity
```

## Scenario: Heartbeat Metrics Contract

### 1. Scope / Trigger

- Trigger: changing the heartbeat payload between C++ Engine and Go Control Plane, adding or removing metrics fields, or changing field semantics.
- This contract spans C++ `HeartbeatReporter::BuildHeartbeatPayload()`, Go `HeartbeatRequest` DTO, Go `EdgeNodeMetrics` model, and frontend service types.

### 2. Signatures

```go
// Go HeartbeatRequest (extended)
type HeartbeatRequest struct {
    // Legacy fields (preserved for backward compatibility)
    CPUUsage     float64 `json:"cpu_usage"`
    MemoryUsage  float64 `json:"memory_usage"`
    Uptime       int64   `json:"uptime"`
    CurrentLoad  int     `json:"current_load"`
    HardwareInfo HardwareInfo `json:"hardware_info"`

    // Extended metrics fields
    CPULoad1m     float64            `json:"cpu_load_1m,omitempty"`
    DiskUsage     []DiskUsageEntry   `json:"disk_usage,omitempty"`
    NetRxBytes    int64              `json:"net_rx_bytes,omitempty"`
    NetTxBytes    int64              `json:"net_tx_bytes,omitempty"`
    NetRxSpeed    float64            `json:"net_rx_speed,omitempty"`
    NetTxSpeed    float64            `json:"net_tx_speed,omitempty"`
    Temperature   float64            `json:"temperature,omitempty"`
    ProcessCount  int                `json:"process_count,omitempty"`
    ThreadCount   int                `json:"thread_count,omitempty"`
    WorkerCount   int                `json:"worker_count,omitempty"`
}
```

### 3. Contracts

- Legacy fields (`cpu_usage`, `memory_usage`, `uptime`, `current_load`, `hardware_info`) must remain at their original JSON positions forever. New engines add extended fields alongside legacy fields.
- All extended fields use `omitempty` so old engines that do not send them will not break Go parsing.
- C++ `HeartbeatReporter` uses `nlohmann/json::parse` which tolerates extra fields from Go control commands.
- `DiskUsage` is a JSON array: `[{"path":"/","total":64000000000,"used":32000000000,"percent":50.0}]`
- Network speed fields are computed as deltas between snapshots, in bytes/second.
- All numeric values are flat doubles or ints — no nested structures beyond `disk_usage` and `hardware_info`.

### 4. Cross-Layer Field Mapping

```text
C++ BuildHeartbeatPayload()  →  Go HeartbeatRequest DTO  →  Go EdgeNodeMetrics model  →  Frontend API type
cpu_usage                    →  cpu_usage                 →  CPUUsage                   →  metric "cpu_usage"
memory_usage                 →  memory_usage              →  MemoryUsage                →  metric "memory_usage"
cpu_load_1m                  →  cpu_load_1m               →  CPULoad1m                  →  metric "cpu_load_1m"
disk_usage[]                 →  disk_usage                →  DiskUsage (JSONB)          →  metric "disk_usage"
net_rx_bytes                 →  net_rx_bytes              →  NetRxBytes                 →  metric "net_rx_bytes"
net_tx_bytes                 →  net_tx_bytes              →  NetTxBytes                 →  metric "net_tx_bytes"
net_rx_speed                 →  net_rx_speed              →  NetRxSpeed                 →  metric "net_rx_speed"
net_tx_speed                 →  net_tx_speed              →  NetTxSpeed                 →  metric "net_tx_speed"
temperature                  →  temperature               →  Temperature                →  metric "temperature"
process_count                →  process_count             →  ProcessCount               →  metric "process_count"
thread_count                 →  thread_count              →  ThreadCount                →  metric "thread_count"
worker_count                 →  worker_count              →  WorkerCount                →  metric "worker_count"
```

### 5. Validation & Error Matrix

| Condition | Result |
|-----------|--------|
| Legacy field missing (e.g., old engine) | Heartbeat still accepted, field defaults to 0 |
| Extended field missing | EdgeNodeMetrics field defaults to 0, no error |
| `disk_usage` array with invalid entries | Invalid entries stored as-is in JSONB; consumed by frontend charts as metric "disk_usage" |
| `temperature` from platform without thermal sensor | Reports 0; frontend should handle zero as "N/A" |

### 6. Good/Base/Bad Cases

- Good: New engine sends all extended fields; Go stores them and displays in frontend charts.
- Base: Old engine sends only legacy fields; Go stores 0 for extended fields; frontend shows "no data" for missing metrics.
- Bad: Extended field naming conflicts with legacy field names or uses reserved JSON keywords.

### 7. Wrong vs Correct

Wrong: Removing a legacy field or changing its JSON key.

Correct: Adding extended fields with `omitempty` while keeping all legacy fields.

## Algorithm Package Layout<br><br>

Algorithm packages live under:

```text
algorithms/<algorithm_name>/<version>/
```

Expected package contents include:

- `algo_meta.yaml`
- `label_map.json`
- `models/`
- `src/`
- `CMakeLists.txt` or `build.sh`
- `test.sh`
- `README.md`

The metadata file is the control-plane parsing entrypoint. Keep algorithm id, version, domain, capabilities, result schema, parameter schema, platform support, and package integrity fields synchronized with backend validation and frontend upload/deploy UI.

## C ABI

The dynamic library contract is defined in `engine/include/algo/abi_contract.h`. Algorithm libraries must export C ABI functions with `extern "C"`, return stable error codes, keep result memory allocation/freeing paired, and catch C++ exceptions before crossing the ABI boundary.

Reference files:
- `engine/include/algo/abi_contract.h`
- `engine/include/algo/algo_instance.h`
- `engine/include/algo/algo_manager.h`
- `engine/include/algo/algorithm_downloader.h`

Do not let algorithms depend on Go process memory, Go private types, or frontend-only assumptions. Communication happens through ABI structs, JSON config, labels, and protocol payloads.

## Result JSON

Inference result JSON must be stable and versionable. Keep field names aligned with Go DTO/database mappings and frontend expectations. Fields such as `category_code`, bbox, confidence, track id, timestamp, ROI, and evidence path need clear units and coordinate systems.

If label maps, category codes, thresholds, input dimensions, or NMS settings change, update `algo_meta.yaml`, `label_map.json`, Go validation/mapping, and frontend display logic together.
