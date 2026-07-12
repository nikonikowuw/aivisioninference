# Backend Quality Guidelines

## Before Editing

Read the relevant `.rules/golang-pro.md` section before Go changes. Use CodeGraph first when locating symbols in this repository because `.codegraph/` is present.

## Context And Lifecycle

Every service and repository method involved in a request should receive `context.Context` first. Goroutines need an explicit lifecycle. Long-running services such as `SystemService` keep a cancel function and expose `Close()`.

Reference files:
- `app/internal/service/system.go`
- `app/internal/service/edge_node.go`

## API And Swagger

New or changed HTTP handlers need:

- DTO structs with validation tags.
- Swagger annotations beside the handler.
- Route registration under `/api/v1` unless it is a documented health/static/webhook exception.
- Uniform `response` package output.

Run `cd app && make swag` after annotation changes.

## Dependency Graph

When adding backend objects:

- Repository/service/handler providers go through `app/internal/router/deps.go` and `wire.go`.
- App-level dependencies go through `app/internal/server/`.
- Regenerate with `cd app && make wire`.

Do not edit `wire_gen.go` manually.

## Tests

Use focused tests for shared utilities, protocol matching, converters, repositories with meaningful query logic, and services with state transitions. Existing examples include:

- `app/internal/pkg/mqttmux/mux_test.go` for MQTT topic matching.
- `app/internal/pkg/controlproto/` tests for protocol parsing/conversion.
- `app/internal/service/*_test.go` for service-level behavior where present.

Run a focused package test first, then `cd app && make unit-test` for broader validation when the change affects shared behavior.

## Backend Anti-Patterns

- Adding business rules to repositories.
- Calling external systems from handlers instead of services.
- Skipping Wire because a direct constructor call is faster to write.
- Adding a model without a migration entry and necessary indexes.
- Using raw SQL with concatenated user input.
- Returning Chinese or English display strings directly from low-level packages when an error code should be translated.
