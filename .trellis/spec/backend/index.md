# Go Control Plane Specs

These specs cover `app/`, the Gin/GORM/Redis control plane. The backend owns API handling, RBAC, task orchestration, storage adapters, MQTT command/response adaptation, edge-node heartbeat ingestion, and persistence. It does not own video decoding, frame-level inference, or algorithm execution; those belong to `engine/` and `algorithms/`.

## Guides

| Guide | Use When |
|-------|----------|
| [Directory Structure](./directory-structure.md) | Adding or moving Go packages, handlers, services, repositories, tasks, or generated code |
| [Database Guidelines](./database-guidelines.md) | Adding models, GORM queries, migrations, indexes, transactions, or JSONB/pgvector data |
| [Error Handling](./error-handling.md) | Returning API errors, translating messages, wrapping internal failures, or handling middleware failures |
| [Logging Guidelines](./logging-guidelines.md) | Adding service, task, MQTT, heartbeat, or operational logs |
| [Edge Node Lifecycle](./edge-node-lifecycle.md) | Changing heartbeat, LWT, offline detection, task suspension/recovery, scheduler registration, or lifecycle WebSocket events |
| [Edge Scheduled Tasks](./edge-scheduled-tasks.md) | Modifying scheduled MQTT command dispatch, EdgeNode tags, patrol worker, or task execution records |
| [Terminal Sessions](./terminal-sessions.md) | Modifying SSH pool, terminal session lifecycle, ttyrec recording, WebSocket terminal handler, or frontend terminal components |
| [Quality Guidelines](./quality-guidelines.md) | Before finishing backend changes or touching shared generated/Wire/Swagger flows |

## Local Anchors

- Main server assembly: `app/cmd/server/main.go`, `app/internal/server/`
- Database migration and seed flow: `app/cmd/migrate/main.go`
- Route dependency assembly: `app/internal/router/deps.go`, `app/internal/router/wire.go`
- Request/response boundary: `app/internal/handler/`, `app/internal/dto/`, `app/internal/pkg/response/`
- Business layer: `app/internal/service/`
- Data access layer: `app/internal/repository/`, `app/internal/model/`
- Protocol adapters: `app/internal/pkg/mqttmux/`, `app/internal/pkg/mqttsync/`, `app/internal/pkg/controlproto/`

## Required Checks

- Dependency injection changes: `cd app && make wire`
- API annotation changes: `cd app && make swag`
- Backend validation: `cd app && make unit-test`
- Build-level check: `cd app && make build`
