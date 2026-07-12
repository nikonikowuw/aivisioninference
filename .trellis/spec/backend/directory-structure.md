# Backend Directory Structure

## Ownership Boundaries

`app/` is organized around a strict control-plane layering:

```text
Handler -> Service -> Repository -> Model
   |          |
   v          v
  DTO    Storage / MQTT / Task / WebSocket
```

Keep that dependency direction. Handlers bind and validate input, services enforce business rules and coordinate side effects, repositories perform GORM access, and models define persistence shape.

Reference files:
- `app/internal/handler/edge_node.go` shows handler binding, Swagger annotations, service calls, and `response.OK` / `response.Page` / `response.Err`.
- `app/internal/service/edge_node.go` shows service-owned uniqueness checks, version compatibility, heartbeat updates, WebSocket notification, and async worker ownership.
- `app/internal/repository/edge_node.go` shows repository-scoped GORM queries using `db.WithContext(ctx)`.
- `app/internal/model/algorithm.go` and `app/internal/model/edge_node.go` define persistence models used across service and frontend DTOs.

## Package Roles

- `cmd/server/`: HTTP server entrypoint.
- `cmd/migrate/`: idempotent extension setup, `AutoMigrate`, manual SQL migrations, indexes, and seed data.
- `cmd/gen/`: CRUD generator entrypoint.
- `internal/handler/`: Gin request boundary. Do not put business workflows here.
- `internal/dto/`: request and response structs. Use `validate` tags for request validation and snake_case JSON tags for API fields.
- `internal/service/`: business rules, transactions, orchestration, external adapters, MQTT command flow, task scheduling, and result ingestion.
- `internal/repository/`: GORM data access only.
- `internal/model/`: GORM models, constants, sortable fields, table-level conventions.
- `internal/router/`: route registration and business dependency graph. Add providers here for route-layer services.
- `internal/server/`: application-level dependency graph and lifecycle.
- `internal/task/`: Asynq workers and periodic tasks.
- `internal/pkg/`: shared infrastructure such as errors, response, JWT, validator, WebSocket, ZLM, MQTT mux/sync, and FlatBuffers conversion.
- `pkg/storage/`: local, PostgreSQL large-object, and S3/OSS compatible storage adapters.
- `pkg/gen/`: CRUD generator and source stubs.

## Dependency Injection

Use Wire for new repositories, services, handlers, and infrastructure providers. Application-wide providers live in `app/internal/server/`; route/business providers live in `app/internal/router/`.

Do not manually edit `wire_gen.go`. Do not instantiate a production handler/service in a route registration function as a shortcut. Update `deps.go` / `wire.go` and run `cd app && make wire`.

Reference files:
- `app/internal/router/deps.go` aggregates route dependencies in `RouteDeps` and wires cross-service setters such as SIP runtime injection.
- `app/internal/router/router.go` stores router-level dependencies and middleware factories.

## Generated Code

Files with a generated header, for example `web/src/router/index.tsx` on the frontend and generated CRUD files under `app/`, should not receive long-lived manual edits. Change the generator or put custom behavior in adjacent handwritten files.

## Common Mistakes

- Bypassing `internal/dto` and binding request JSON directly into a GORM model.
- Letting a repository decide permissions, task state transitions, or cross-resource constraints.
- Adding a handler/service/repository without updating Wire.
- Putting video decoding or frame inference logic in Go. The Go layer should issue commands and persist results; the C++ engine executes data-plane work.
