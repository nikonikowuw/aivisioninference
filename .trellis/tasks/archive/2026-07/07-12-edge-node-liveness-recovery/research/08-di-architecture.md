# Dependency Injection & Architecture Overview

## Router Wire Setup

### File
`/Users/niko/dev/go/aivisioninference/app/internal/router/wire.go`

The `InitializeRouteDeps()` function uses Google Wire to construct the dependency graph. Key edge node-related providers:

```go
var repositorySet = wire.NewSet(
    // ...other repos...
    repository.NewEdgeNodeRepository,         // *repository.EdgeNodeRepository
    repository.NewEdgeNodeAlgorithmRepository, // *repository.EdgeNodeAlgorithmRepository
    repository.NewAIVisionTaskRepository,      // *repository.AIVisionTaskRepository
)

var serviceSet = wire.NewSet(
    // ...other services...
    provideEdgeNodeService,     // *service.EdgeNodeService
    provideMqttSyncManager,     // *mqttsync.MqttSyncManager
    provideMqttMux,             // *mqttmux.Mux
    provideEngineMetricsStore,  // *service.EngineMetricsStore
    provideEdgeMqttHandler,     // *handler.EdgeMqttHandler
)

var handlerSet = wire.NewSet(
    // ...other handlers...
    provideEdgeNodeHandler,       // *handler.EdgeNodeHandler
    provideEdgeNodeMiddleware,    // *middleware.EdgeNodeMiddleware
)

func InitializeRouteDeps(db *gorm.DB, rdb *redis.Client, jwtManager *jwt.Manager, 
    hub *ws.Hub, cfg *Config, scheduler *asynq.Scheduler, 
    mqttClient mqtt.Client) (*RouteDeps, error) { ... }
```

## Provider Functions

### File
`/Users/niko/dev/go/aivisioninference/app/internal/router/deps.go`

### EdgeNodeService Provider (line 459)
```go
func provideEdgeNodeService(
    nodeRepo, nodeAlgoRepo, algoPackageRepo, taskRepo, 
    deviceRepo, smartRecordRepo, jwtManager, fileStorage, cfg, hub,
) *service.EdgeNodeService {
    svc := service.NewEdgeNodeService(...)
    svc.SetVersionConfig(cfg.Engine.MinCompatibleVersion, cfg.Engine.VersionCheckEnabled)
    return svc
}
```

### EdgeMqttHandler Provider (line 501)
```go
func provideEdgeMqttHandler(
    nodeSvc, syncManager, taskClient, rdb, hub, metricsStore,
) *handler.EdgeMqttHandler {
    return handler.NewEdgeMqttHandler(nodeSvc, syncManager, taskClient, rdb, hub, metricsStore)
}
```

## RouteDeps Struct
```go
type RouteDeps struct {
    // ...
    EdgeNodeHandler    *handler.EdgeNodeHandler
    EdgeNodeMiddleware *middleware.EdgeNodeMiddleware
    EdgeNodeSvc        *service.EdgeNodeService
    EdgeMqttHandler    *handler.EdgeMqttHandler
    MqttMux            *mqttmux.Mux
    EngineMetricsStore *service.EngineMetricsStore
}
```

## MQTT Server Setup

### File
`/Users/niko/dev/go/aivisioninference/app/internal/server/deps.go` (line 98+)

```go
func provideMqttClient(cfg *config.Config) (mqtt.Client, error) {
    opts := mqtt.NewClientOptions()
    // ... broker, credentials ...
    opts.SetAutoReconnect(true)
    opts.SetCleanSession(false)
    // NOTE: No LWT configuration on server side (LWT is only on engine side)
    
    client := mqtt.NewClient(opts)
    if token := client.Connect(); token.Wait() && token.Error() != nil {
        return nil, fmt.Errorf("MQTT connection failed: %w", token.Error())
    }
    return client, nil
}
```

### File
`/Users/niko/dev/go/aivisioninference/app/internal/server/mqtt_server.go`

```go
func (s *MqttServer) Start() error {
    s.mux.Register("aivision/edge/+/status/heartbeat", s.handler.HandleHeartbeat)
    s.mux.Register("aivision/edge/+/status/lifecycle", s.handler.HandleLifecycle)
    s.mux.Register("aivision/edge/+/response/+", s.handler.HandleStreamStatus)
    s.mux.Register("aivision/edge/+/event/inference", s.handler.HandleInferenceResult)
    s.mux.Register("aivision/edge/+/event/metrics", s.handler.HandleEngineMetrics)
    
    // Subscribe to aivision/edge/# (QoS 1)
    s.client.Subscribe("aivision/edge/#", 1, func(c mqtt.Client, msg mqtt.Message) {
        s.mux.Dispatch(msg)
    })
}
```

## Asynq Setup

### File
`/Users/niko/dev/go/aivisioninference/app/internal/router/router.go`

```go
func NewAsynqMux(db *gorm.DB, rdb *redis.Client, cfg *Config, 
    mqttClient mqtt.Client, syncManager *mqttsync.MqttSyncManager) *asynq.ServeMux {
    
    // ... create repos, services ...
    
    // Edge Node Status Checker
    hub := ws.NewHub()
    edgeNodeStatusTask := task.NewEdgeNodeStatusTask(nodeRepo, aiTaskRepo, hub, cfg.Engine.HeartbeatTimeoutSec)
    edgeNodeStatusTask.RegisterHandlers(mux)
    
    // Edge Node State Reconciliation Worker
    edgeStateWorker := task.NewEdgeStateWorker(aiTaskRepo, nodeRepo, rdb, engineClient)
    mux.HandleFunc(task.TaskReconcileEdgeState, edgeStateWorker.HandleReconcileEdgeState)
    
    return mux
}

func NewAsynqScheduler(rdb *redis.Client) *asynq.Scheduler {
    return task.NewScheduler(rdb)
}
```

### File
`/Users/niko/dev/go/aivisioninference/app/internal/task/server.go`

```go
func RegisterPeriodicTasks(scheduler *asynq.Scheduler) {
    scheduler.Register("*/5 * * * *", asynq.NewTask(TypeDeviceStatusCheck, nil))
    scheduler.Register("* * * * *", asynq.NewTask(TypeAIVisionTaskPatrol, nil))
    // ❌ TypeEdgeNodeStatusCheck NOT registered here
}
```

## Server Startup

### File
`/Users/niko/dev/go/aivisioninference/app/internal/server/deps.go`

```go
func provideAsynqScheduler(rdb *redis.Client) *asynq.Scheduler {
    return router.NewAsynqScheduler(rdb)
}

func provideAsynqMux(db *gorm.DB, rdb *redis.Client, cfg *router.Config, client mqtt.Client) *asynq.ServeMux {
    syncManager := mqttsync.NewMqttSyncManager(rdb)
    return router.NewAsynqMux(db, rdb, cfg, client, syncManager)
}
```

## EngineMetricsStore

### File
`/Users/niko/dev/go/aivisioninference/app/internal/service/engine_metrics_store.go`

Stores per-node metrics snapshots from MQTT `aivision/edge/{id}/event/metrics` topic. Used for:
- Real-time engine monitoring (system page)
- Media scheduling decisions (e.g., `GetFreshNodeMediaMetrics()`)
- History trends via `HistoryBuffer`

## Architecture Diagram

```
Browser/API Clients
    │
    ├── HTTP (Gin) ─── EdgeNodeHandler ─── EdgeNodeService ─── Repository (GORM)
    │                                                              │
    │                                                         [PostgreSQL]
    │
    ├── WebSocket ─── ws.Hub ─── EdgeNodeService / EdgeNodeStatusTask
    │
    └── [Admin UI]
    
Edge Node (C++ Engine)
    │
    ├── MQTT Heartbeat ─── MqttServer.HandleHeartbeat() ─── EdgeNodeService
    ├── MQTT Lifecycle (LWT) ─── MqttServer.HandleLifecycle()
    ├── MQTT Inference ─── MqttServer.HandleInferenceResult()
    ├── MQTT Metrics ─── MqttServer.HandleEngineMetrics() ─── EngineMetricsStore
    │
    └── MQTT Commands ←── MqttEngineClient (request-response via Redis)
    
Background (Asynq)
    ├── [NOT RUNNING] edge_node:status_check ─── EdgeNodeStatusTask
    ├── aivision:patrol ─── AIVisionTaskPatrol
    ├── device:status_check ─── DeviceStatusHandler
    └── edge_node:reconcile_state ─── EdgeStateWorker
```
