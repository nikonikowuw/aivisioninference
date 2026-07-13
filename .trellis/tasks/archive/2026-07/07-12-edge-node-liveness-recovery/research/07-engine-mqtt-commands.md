# Engine MQTT Commands

## Overview
The server communicates with edge inference engines via MQTT commands published to topic `aivision/edge/{node_id}/cmd/{command}`. The response is correlated via a `trace_id` using Redis-based Pub/Sub (MqttSyncManager).

---

## MqttEngineClient

### File
`/Users/niko/dev/go/aivisioninference/app/internal/service/mqtt_engine_client.go`

### Topic Generation
```go
func edgeCommandTopic(nodeID, command string) string {
    return fmt.Sprintf("aivision/edge/%s/cmd/%s", nodeID, command)
}
```

---

## Supported Commands

| Command | Method | Description | Timeout |
|---------|--------|-------------|---------|
| `start_stream` | `StartStream()` | Start a video stream with optional inference | 5s |
| `stop_stream` | `StopStream()` | Stop a video stream | 5s |
| `start_playback` | `StartPlayback()` | Start video playback (historical) | 5s |
| `stop_playback` | `StopPlayback()` | Stop video playback | 5s |
| `stream_status` | `GetStreamStatus()` | Query stream state | 5s |
| `algo_warmup` | `WarmupAlgorithm()` | Pre-load algorithm runtime | 10s |
| `face_library` | `UpdateFaceLibrary()` | Push face library snapshot | 10s |
| `face_embedding` | `ExtractFaceEmbedding()` | Extract embedding from image | 30s |

### Special Topic (not node-specific):
| Topic | Method | Description | Timeout |
|-------|--------|-------------|---------|
| `aivision/edge/self_check/cmd` | `StartSelfCheck()` | Verify algorithm loads correctly | 5min |

---

## EngineClient Interface

### File
`/Users/niko/dev/go/aivisioninference/app/internal/service/ipc_engine_client.go`

```go
type EngineClient interface {
    StartStream(ctx context.Context, req StreamStartRequest) (StreamInfo, error)
    StopStream(ctx context.Context, nodeID, deviceID string) error
    StartPlayback(ctx context.Context, req StreamStartRequest) (string, error)
    StopPlayback(ctx context.Context, nodeID, deviceID string) error
    GetStreamStatus(ctx context.Context, nodeID, deviceID string) (StreamStatus, error)
    StartSelfCheck(ctx context.Context, downloadURL, token, algoName, version string) error
    WarmupAlgorithm(ctx context.Context, nodeID, algoName, algoVersion string) error
    UpdateFaceLibrary(ctx context.Context, nodeID, algoName string, faceLibraryJSON []byte) error
    ExtractFaceEmbedding(ctx context.Context, nodeID, algoName, algoVersion string, imageBytes []byte) (FaceEmbeddingResult, error)
}
```

### Implementations
1. **`MqttEngineClient`** — Production implementation uses MQTT + Redis sync
2. **`MockEngineClient`** — Test/local dev stub

---

## Request-Response Pattern

All commands follow the same pattern:

```go
// 1. Generate trace_id
traceID := uuid.New().String()

// 2. Marshal payload with trace_id
payload, _ := json.Marshal(map[string]string{
    "trace_id":  traceID,
    "device_id": deviceID,
})

// 3. Publish to command topic
topic := edgeCommandTopic(nodeID, command)
token := c.mqttClient.Publish(topic, 1, false, payload)
token.Wait()

// 4. Wait for response via Redis Pub/Sub
respPayload, err := c.syncManager.Wait(ctx, traceID, timeout)
```

### Response Topic
```
aivision/edge/{node_id}/response/{command}
```
Handled by `EdgeMqttHandler.HandleStreamStatus()` which parses the `trace_id` and resolves the pending wait via `syncManager.Resolve()`.

---

## MqttSyncManager

### File
`/Users/niko/dev/go/aivisioninference/app/internal/pkg/mqttsync/manager.go`

Uses Redis Pub/Sub for request-response correlation:
- `Wait(ctx, traceID, timeout)` → subscribes to Redis channel `trace:{traceID}`, waits for message
- `Resolve(ctx, traceID, payload)` → publishes payload to `trace:{traceID}` channel

---

## Command Dispatch Flow (StartStream Example)

```
Browser/API → AIVisionTaskService.StartTask()
  → MqttEngineClient.StartStream()
    → Publish: aivision/edge/{nodeID}/cmd/start_stream (QoS 1)
    → Wait: Redis Pub/Sub channel trace:{traceID} (5s timeout)
    
Engine receives command → starts stream
  → Publish: aivision/edge/{nodeID}/response/start_stream (QoS 1)
    with JSON: {"trace_id": "...", "status": "running", "play_url": "..."}

Server MqttServer dispatch:
  → mux.Dispatch() matches "aivision/edge/+/response/+"
  → EdgeMqttHandler.HandleStreamStatus()
    → syncManager.Resolve(traceID, payload)
      → Publishes to Redis channel trace:{traceID}
        → MqttEngineClient's Wait() unblocks
```

---

## Edge Node Subscriptions

Engine subscribes to (from C++ code in `mqtt_control_plane.cpp`):
```
aivision/edge/{node_id}/cmd/#   (QoS 1)
aivision/edge/self_check/cmd    (QoS 1)
```

---

## What's Missing / Broken

1. **No "stop_all" command** — There's no way to tell an engine to stop all streams at once. The server must iterate and call `StopStream` per device.

2. **No "recover" or "restart" command** — No EngineClient method exists to ask an engine to restart or recover.

3. **No "ping" command** — No lightweight connectivity check. The heartbeat serves this purpose but is a full state sync.

4. **No batch commands** — All commands are per-device. Starting 20 streams means 20 separate MQTT publish+wait cycles.

5. **`Wait()` timeout is hardcoded** — Each command has its own timeout (5s, 10s, 30s, 5min). These are not configurable.

6. **No command retry** — If a `Publish()` succeeds but `Wait()` times out, there's no retry. The server assumes failure.

7. **No queue depth visibility** — The engine could be overwhelmed with commands, but there's no backpressure mechanism.
