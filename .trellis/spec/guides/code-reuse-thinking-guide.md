# Code Reuse Thinking Guide

## Before Writing New Code

Search for an existing local pattern before creating a helper, adapter, query builder, DTO, route mapping, protocol converter, or UI wrapper. In this repo, start with CodeGraph when available, then use `rg`.

Good searches:

```bash
codegraph explore "edge node heartbeat service repository"
rg "BuildQuery|buildQuery|Paginate|OrderBy|response.OK|request<" app web engine
rg "StartStream|StopStream|InferenceResult|EngineMetrics" app engine proto
```

## Backend Reuse Targets

Prefer existing backend helpers:

- API responses: `app/internal/pkg/response/`
- Business errors and i18n: `app/internal/pkg/errors/`, `app/internal/pkg/i18n/`
- Query pagination and ordering: `app/internal/pkg/scopes/`
- Validation: `app/internal/pkg/validator/`
- MQTT topic dispatch: `app/internal/pkg/mqttmux/`
- MQTT sync/response management: `app/internal/pkg/mqttsync/`
- FlatBuffers/protocol conversion: `app/internal/pkg/controlproto/`

If several repository methods differ only by filters, use a private shared query builder like `findNodesByAlgoJoin` in `app/internal/repository/edge_node.go`.

## Frontend Reuse Targets

Prefer existing frontend primitives:

- API and auth behavior: `web/src/services/api.ts`
- Typed feature services: `web/src/services/`
- Pagination/filter/date/WebSocket hooks: `web/src/hooks/`
- Video playback and stream sharing: `web/src/components/VideoPlayer/`
- Common admin UI: `web/src/components/search-bar/`, `pagination/`, `confirm-dialog/`, `empty/`, `skeleton/`
- Chakra theme tokens: `web/src/theme/`

Do not add direct `fetch`, a new toast/error parser, or a page-local query-string builder without checking the service layer first.

## Engine Reuse Targets

Prefer existing engine contracts:

- Stream lifecycle: `engine/include/pipeline/pipeline_manager.h`
- Hardware abstraction: `engine/include/pipeline/hal.h`
- Worker queues and buffers: `engine/include/pipeline/ring_queue.h`, `hw_buffer.h`, `worker_pool.h`
- Algorithm ABI and instance lifecycle: `engine/include/algo/abi_contract.h`, `algo_instance.h`, `algo_manager.h`
- Response routing: `engine/include/response_router.h`
- Metrics/heartbeat reporting: `engine/include/monitor/`

Do not create a parallel stream map, algorithm loader, or command response path unless the existing owner cannot represent the new behavior.

## Duplication Warning Signs

- Two layers define the same status strings independently.
- Frontend interfaces repeat backend DTOs with slightly different optional fields.
- A service and repository both enforce the same business rule.
- MQTT JSON and FlatBuffers schema evolve separately.
- Algorithm `label_map.json`, `algo_meta.yaml`, Go validation, and frontend display names drift.

When any of these appear, consolidate the source of truth or document the intentional boundary.
