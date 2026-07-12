# Database Guidelines

## ORM And Context

GORM is the persistence layer. Repository methods must accept `ctx context.Context` first and use `r.db.WithContext(ctx)` for every query. Handlers pass `c.Request.Context()` into services, and services pass the same context into repositories.

Reference files:
- `app/internal/repository/edge_node.go`
- `app/internal/repository/algorithm_package.go`
- `app/internal/pkg/scopes/`

Avoid `context.Background()` inside request-driven handler/service/repository code. It is acceptable for process lifecycle, CLI commands, Asynq roots, tests, and startup probes.

## Models And Migrations

Add new persistent models under `app/internal/model/` and register them in `app/cmd/migrate/main.go`. That migration entrypoint also owns PostgreSQL extensions, partial indexes, JSONB type fixes, pgvector indexes, partition setup, and seed flow.

The migration file is intentionally idempotent:
- It uses `CREATE EXTENSION IF NOT EXISTS vector`.
- It wraps type/index repairs in guarded SQL blocks.
- It creates partial indexes such as `idx_edge_nodes_name ON edge_nodes (name) WHERE deleted_at IS NULL`.
- It calls `AutoMigrate` only after manual incompatible type conversions.

When a GORM `AutoMigrate` cannot express a safe schema transition, add an explicit guarded SQL step before or after `AutoMigrate`.

## Query Patterns

Use shared scopes for pagination and ordering instead of rewriting pagination in each repository. `EdgeNodeRepository.List` combines request filter scopes with `scopes.Paginate` and `scopes.OrderBy`.

Keep shared query builders private to repositories when several methods differ only by filters. `findNodesByAlgoJoin` in `app/internal/repository/edge_node.go` centralizes online/enabled/installed node filtering for algorithm-capability queries.

Use GORM parameter binding for user-controlled input:

```go
Where("edge_node_algorithms.algo_package_id = ?", algoPackageID)
```

Do not concatenate user input into SQL strings.

## Transactions

Services own transaction boundaries because they know the business workflow. Repositories may expose focused transaction helpers or `ForUpdate` lookup methods, but should not decide the workflow.

Reference files:
- `app/internal/repository/edge_node.go` has `FindByIDForUpdate` with `clause.Locking{Strength: "UPDATE"}`.
- `app/internal/repository/edge_node.go` updates heartbeat fields with an allowlist to avoid concurrent write conflicts.

Be careful with transaction helpers that accept only `context.Context`; if repository calls inside the callback do not receive the transaction DB, the transaction may not protect the intended operations. Prefer explicit transaction-aware repository methods or a repository instance bound to `tx` for multi-step writes.

Media preview admission is one such multi-step transaction. Lock schedulable nodes in stable ID order, delete expired pending reservations, reuse an effective `stream_key` reservation, sum one `preview_slots` unit per effective reservation, and create the selected pending lease before releasing the transaction. Capacity and live usage come from the fresh Engine snapshot, not persistent edge-node resource-vector columns.

## JSONB, Vector, And Protocol Data

Use typed DTO/conversion code at boundaries where JSONB, pgvector, or FlatBuffers data crosses layers. Do not assemble FlatBuffers payloads or parse algorithm result JSON directly in HTTP handlers.

Reference files:
- `proto/flatbuf/README.md` documents `InferTask`, `AlgorithmPackage`, `SmartRecord`, and result-message mappings.
- `app/internal/pkg/controlproto/` contains protocol conversion and generated FlatBuffers code.

## Verification

- New or changed model: update `app/cmd/migrate/main.go`.
- New sortable/filterable list: add model sortable fields or request filter scopes and test edge cases.
- Schema or query risk: run focused repository/service tests, then `cd app && make unit-test`.
