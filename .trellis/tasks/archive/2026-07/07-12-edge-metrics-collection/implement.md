# System Metrics Collection & Dashboard — Implementation Plan

## Order

1. **C++ DeviceMonitor probes** — extend disk/net/load/process/temp collection
2. **C++ MetricsFlattener** — new module to flatten DeviceSnapshot for heartbeat
3. **C++ HeartbeatReporter** — extend BuildHeartbeatPayload with new fields
4. **Go Model + Migration** — EdgeNodeMetrics model + auto-migrate
5. **Go Repository** — EdgeNodeMetricsRepository with aggregation query
6. **Go DTO Extension** — add new fields to HeartbeatRequest
7. **Go Service** — extend HandleHeartbeat to write metrics; new metrics query service
8. **Go Handler + Router** — metrics query API + overview API
9. **Go Wire** — register new providers
10. **Go Metrics Retention Cron** — Asynq cron task for data cleanup
11. **React Overview Page** — stat cards + status distribution chart
12. **React EdgeNodeList** — add real-time metric columns + WS integration
13. **React EdgeNodeDetail** — add metrics charts tab

## Validation Gates

Each step must pass before next:
- `make engine` — C++ compiles
- `make server` — Go compiles (also `make wire` if DI changes)
- `make web` — React compiles
- Manual test with running engine + control plane
