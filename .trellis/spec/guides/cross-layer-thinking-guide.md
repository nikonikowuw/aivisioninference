# Cross-Layer Thinking Guide

## Why This Matters Here

A single product change can cross:

```text
React page -> frontend service type -> Go DTO/handler -> service/repository/model
          -> MQTT JSON command -> C++ dispatcher/engine -> FlatBuffers event
          -> Go result ingestion -> database -> WebSocket/frontend display
```

Most regressions happen when only one side of that contract is updated.

## Contract Checklist

For API or model changes, check:

- `app/internal/model/`
- `app/internal/dto/`
- `app/internal/handler/`
- `app/internal/service/`
- `app/cmd/migrate/main.go`
- `web/src/services/`
- `web/src/views/admin/`
- Locale files under `web/src/locales/`

For engine/protocol changes, check:

- Go MQTT command builders and response handlers in `app/internal/service/` and `app/internal/pkg/controlproto/`
- C++ MQTT and dispatch code in `engine/src/mqtt_control_plane.cpp`, `engine/src/command_dispatcher.cpp`, and `engine/src/response_router.cpp`
- FlatBuffers schema in `proto/flatbuf/`
- Generated Go and C++ FlatBuffers output
- Frontend service/page expectations when status or result fields are displayed

For algorithm package changes, check:

- `algorithms/<name>/<version>/algo_meta.yaml`
- `label_map.json`
- C ABI implementation and `engine/include/algo/abi_contract.h`
- Go package upload/validation/deployment services
- Frontend algorithm package upload/deploy UI

## Status And Field Naming

Backend JSON uses snake_case. Frontend service types usually mirror backend snake_case fields. Engine C++ types use C++ naming and protocol payloads bridge the naming. Be explicit about the mapping; do not rely on implicit frontend casts or ad hoc string conversions.

When adding status values, update every consumer:

- Go model constants
- Service state transitions
- DB seed/migration data if persisted
- Frontend badges/filters/translations
- MQTT command/response handling if edge nodes use it
- Engine command/status mapping if data-plane behavior changes

## Compatibility

FlatBuffers schema evolution must preserve compatibility: add optional fields where possible, do not reorder fields, and deprecate rather than delete. MQTT JSON changes should keep old fields during rolling upgrades when edge nodes may run older engine versions.

For edge nodes and algorithm packages, consider version compatibility. `EdgeNodeService` already has version-check configuration, and engine/package compatibility is part of deployment safety.

## Verification Strategy

Start with focused checks for the touched layer, then add cross-layer validation:

- Backend only: `cd app && make unit-test`
- Frontend only: `cd web && npm run build && npm run test`
- Engine only: `cd engine && make test`
- Protocol spanning Go/C++: regenerate FlatBuffers, build app and engine, run focused protocol tests
- End-to-end task lifecycle: verify command send, engine response, result ingestion, WebSocket/UI update, and cleanup

## Common Cross-Layer Bugs

- API returns `page_size`, frontend expects `pageSize`.
- Backend adds an error code without frontend translations.
- Engine emits a new status that frontend badges treat as unknown.
- Algorithm metadata changes but Go upload validation still expects the old shape.
- FlatBuffers schema changes without regenerating one language target.
- A route menu code changes without updating `menu:<code>` locale keys and router mapping.
