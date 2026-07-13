# MQTT Last Will and Testament (LWT)

## Overview
The MQTT LWT mechanism provides the only automatic node-offline detection that currently works in production. It relies on the MQTT broker's retained-message feature.

---

## Server-Side Handler

### File
`/Users/niko/dev/go/aivisioninference/app/internal/handler/edge_mqtt_handler.go`

### Function
```go
func (h *EdgeMqttHandler) HandleLifecycle(msg mqtt.Message)
```

### Topic
```
aivision/edge/{node_id}/status/lifecycle
```
Registered in `/Users/niko/dev/go/aivisioninference/app/internal/server/mqtt_server.go`:
```go
s.mux.Register("aivision/edge/+/status/lifecycle", s.handler.HandleLifecycle)
```

### Handler Logic
```go
func (h *EdgeMqttHandler) HandleLifecycle(msg mqtt.Message) {
    topicParts := strings.Split(msg.Topic(), "/")
    nodeID := topicParts[2]

    payloadStr := strings.TrimSpace(string(msg.Payload()))
    if payloadStr == "offline" {
        var req dto.UpdateEdgeNodeRequest
        req.Status = "offline"
        h.nodeSvc.Update(context.Background(), nodeID, req)
    }
}
```

---

## Engine-Side (C++)

### File
`/Users/niko/dev/go/aivisioninference/engine/src/mqtt_control_plane.cpp`

### LWT Setup (lines 149-167)
```cpp
// Setup Last Will and Testament (LWT)
std::string lwt_topic = "aivision/edge/" + impl_->node_id_ + "/status/lifecycle";
mqtt::will_options will(lwt_topic, std::string("offline"), 1, true);  // QoS 1, retained
connOpts.set_will(will);
```

### Lifecycle Publishing

**On connect (after successful connection):**
```cpp
impl_->client_->publish(lwt_topic, "online", 1, true)->wait();  // QoS 1, retained
```

**On graceful disconnect (Stop method):**
```cpp
impl_->client_->publish(lwt_topic, "offline", 1, true)->wait();
impl_->client_->disconnect()->wait();
```

### How LWT Works
1. Engine connects with a **retained** LWT message: `aivision/edge/{id}/status/lifecycle` = "offline"
2. On successful connect, engine publishes retained **"online"** to the same topic (overwrites the LWT)
3. If the engine disconnects **unexpectedly** (crash, network loss), the broker publishes the LWT message: "offline"
4. On graceful shutdown, engine publishes "offline" itself before disconnecting

---

## Current Gaps

### 1. Task Suspension NOT Performed
The `HandleLifecycle` handler only calls `nodeSvc.Update()` to set the status to "offline". It does **NOT** suspend the node's running tasks. Compare with `EdgeNodeStatusTask.suspendNodeTasks()` which calls:
```go
h.taskRepo.SetErrorReason(ctx, task.ID, reason)  // → status = "suspended"
```

This means:
- If a node dies suddenly → LWT fires → node status set to "offline" ✅
- But the running tasks remain in "running" status ❌
- They will never be picked up for re-scheduling

### 2. No "online" Lifecycle Handling
The handler ignores `payloadStr == "online"` entirely. When the engine reconnects:
- The retained "offline" LWT gets overwritten by "online"
- But the server does nothing with the "online" message
- The node status remains "offline" until the next heartbeat updates it to "online"
- This works because heartbeat sets status="online", but the lifecycle event could be used for early recovery

### 3. No Node Validation
The MQTT handler trusts whatever `nodeID` comes in the topic. There's no JWT or HMAC validation of the lifecycle message. A malicious actor could publish `aivision/edge/fake-node/status/lifecycle` with "offline" to disrupt operations (limited DoS).

### 4. No Reconnection Backoff
The LWT fires for every unexpected disconnect, including transient network blips. A node that flips between connected/disconnected rapidly will cause repeated status flips.

### 5. No Broker-Level LWT Verification
The Go server does not verify that the LWT message actually came from the broker (as opposed to a third party publishing to the same topic).

---

## MQTT Topic Architecture

### Server subscribes to:
```
aivision/edge/#  (QoS 1)
```
### Topics dispatched:

| Topic Pattern | Handler | Direction |
|--------------|---------|-----------|
| `aivision/edge/{id}/status/heartbeat` | `HandleHeartbeat` | Node → Server |
| `aivision/edge/{id}/status/lifecycle` | `HandleLifecycle` | Node → Server (LWT) |
| `aivision/edge/{id}/response/+` | `HandleStreamStatus` | Node → Server |
| `aivision/edge/{id}/event/inference` | `HandleInferenceResult` | Node → Server |
| `aivision/edge/{id}/event/metrics` | `HandleEngineMetrics` | Node → Server |

### Server publishes to:
```
aivision/edge/{id}/cmd/{command}  (via MqttEngineClient)
```

**Engine subscribes to:**
```
aivision/edge/{id}/cmd/#
aivision/edge/self_check/cmd
```
