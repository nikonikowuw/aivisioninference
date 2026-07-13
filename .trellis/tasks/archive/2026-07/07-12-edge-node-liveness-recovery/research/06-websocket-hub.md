# WebSocket Hub

## Overview
The WebSocket Hub provides real-time bidirectional communication between the server and browser clients. It's used to broadcast node status changes, task status updates, and inference results.

---

## Hub Implementation

### File
`/Users/niko/dev/go/aivisioninference/app/internal/pkg/ws/hub.go`

### Type
```go
type Hub struct {
    clients           map[string]map[*Client]bool  // userID → clients
    broadcast         chan *Message                 // incoming broadcast messages
    register          chan *Client                  // new client registrations
    unregister        chan *Client                  // client disconnections
    lastInferenceSent sync.Map                      // deviceID → time.Time (rate limiting)
    mu                sync.RWMutex
}
```

### Message Type
```go
type Message struct {
    Type     string      `json:"type"`
    Payload  interface{} `json:"payload"`
    DeviceID string      `json:"-"`  // used for inference rate limiting; set before Broadcast
}
```

---

## Hub Usage in Edge Node Context

### 1. `edge-node-status` Event
**Broadcast from:** `EdgeNodeService.HandleHeartbeat()` (service/edge_node.go)

Triggered when:
- Heartbeat received (node comes online)
- Node status changes (error, load change)

```go
s.hub.Broadcast(&ws.Message{
    Type: "edge-node-status",
    Payload: map[string]interface{}{
        "node_id":        id,
        "status":         node.Status,
        "current_load":   node.CurrentLoad,
        "engine_version": node.EngineVersion,
        "last_heartbeat": node.LastHeartbeat,
    },
})
```

**Broadcast from:** `EdgeNodeStatusTask.broadcastNodeOffline()` (task/edge_node_status.go)

Triggered when:
- Periodic check detects timeout (but this never runs — see research/03)

```go
h.hub.Broadcast(&ws.Message{
    Type: "edge-node-status",
    Payload: map[string]interface{}{
        "node_id": nodeID,
        "status":  model.NodeStatusOffline,
    },
})
```

### 2. `task-status` Event
**Broadcast from:** `EdgeNodeService.HandleHeartbeat()` (on task resume)

```go
s.hub.Broadcast(&ws.Message{
    Type: "task-status",
    Payload: map[string]interface{}{
        "task_id":      task.ID,
        "status":       model.TaskStatusRunning,
        "error_reason": "",
        "node_id":      id,
    },
})
```

**Broadcast from:** `EdgeNodeStatusTask.suspendNodeTasks()` (never reached)

```go
h.hub.Broadcast(&ws.Message{
    Type: "task-status",
    Payload: map[string]interface{}{
        "task_id":      task.ID,
        "status":       model.TaskStatusSuspended,
        "error_reason": reason,
        "node_id":      node.ID,
    },
})
```

### 3. `inference` Event
**Broadcast from:** `EdgeMqttHandler.HandleInferenceResult()`

```go
h.hub.Broadcast(&ws.Message{
    Type:     "inference",
    DeviceID: params.DeviceID,
    Payload: map[string]interface{}{
        "device_id":  params.DeviceID,
        "task_id":    params.TaskID,
        "detections": params.Detections,
    },
})
```

Rate-limited to ~30fps per device in the Hub's event loop.

### 4. `edge-node-algo-status` Event
**Broadcast from:** `EdgeNodeService.HandleHeartbeat()` (when pending deployments exist)

```go
s.hub.Broadcast(&ws.Message{
    Type: "edge-node-algo-status",
    Payload: map[string]interface{}{
        "node_id": id,
    },
})
```

---

## Client Management

### Connection
```go
func (h *Hub) HandleConnection(conn *websocket.Conn, userID string) {
    client := newClient(h, conn, userID)
    h.register <- client
    go client.writePump()
    go client.readPump()
}
```

### Per-User Messaging
```go
func (h *Hub) SendToUser(userID string, msg *Message) {
    // Sends to all connections of a specific user
}
```

### Client File
`/Users/niko/dev/go/aivisioninference/app/internal/pkg/ws/client.go` — handles read/write pumps, ping/pong, and send channel.

---

## Dependency Injection

### Wire Setup
**File:** `/Users/niko/dev/go/aivisioninference/app/internal/router/wire.go`

The Hub is created in the top-level app and injected into:
- `EdgeNodeService` (via `NewEdgeNodeService`)
- `EdgeNodeStatusTask` (via `NewEdgeNodeStatusTask`)
- `EdgeMqttHandler` (via `NewEdgeMqttHandler`)

```go
// In top-level app (cmd/server/main.go or server setup):
hub := ws.NewHub()
go hub.Run()

// Then injected to all components
```

### Default Port
`/Users/niko/dev/go/aivisioninference/app/internal/config/config.go`:
```go
v.SetDefault("app.port", 8080)
```
WebSocket typically uses the same port via HTTP upgrade.

---

## What's Missing / Broken

1. **No per-node event channels** — All broadcast events are global. There's no way for a client to subscribe to events for a specific node only. Clients receive all `edge-node-status` events and must filter client-side.

2. **No event persistence** — If no WebSocket client is connected when an event fires, the event is lost. Newly connecting clients don't receive the current state.

3. **No user-specific task notifications** — `SendToUser` exists but edge node events always use `Broadcast` (global). Task status changes are broadcast to ALL users, not just those who manage the affected node.

4. **No reconnection state sync** — When a WebSocket client reconnects, there's no mechanism to send the current state (e.g., "node X is offline").

5. **`broadcast` channel is unbuffered internally** — The channel has a 256 buffer, but if broadcast outpaces client consumption, slow clients are dropped silently.

6. **Edge node status broadcasts are duplicated** — Both the periodic check (if it ran) and the heartbeat handler broadcast `edge-node-status`. The heartbeat handler already broadcasts on every heartbeat, making the check's broadcast redundant.
