# Cross-Cutting Thinking Guides

These guides help avoid mistakes in a multi-runtime system where Go, React, C++, MQTT, FlatBuffers, PostgreSQL, Redis, ZLMediaKit, and algorithm packages share contracts.

## Available Guides

| Guide | Use When |
|-------|----------|
| [Code Reuse Thinking Guide](./code-reuse-thinking-guide.md) | Before creating helpers, constants, protocol converters, route mappings, service wrappers, or duplicated query logic |
| [Cross-Layer Thinking Guide](./cross-layer-thinking-guide.md) | When a change touches API DTOs, frontend service types, database models, MQTT messages, FlatBuffers schema, algorithm metadata, or engine command handling |

## High-Risk Triggers

Read the cross-layer guide when changing:

- `app/internal/model/`, `app/internal/dto/`, or `app/internal/pkg/controlproto/`
- `web/src/services/`, `web/src/router/`, or locale menu keys
- `engine/src/command_dispatcher.cpp`, `engine/src/mqtt_control_plane.cpp`, or `engine/src/response_router.cpp`
- `proto/flatbuf/*.fbs`
- `algorithms/*/*/algo_meta.yaml` or `label_map.json`
- Edge-node heartbeat, algorithm deployment, AI task lifecycle, smart-record result ingestion, or video stream status

Read the code-reuse guide before adding a new helper if a similar pattern exists in:

- `app/internal/pkg/scopes/`
- `app/internal/pkg/response/`
- `app/internal/pkg/errors/`
- `web/src/services/api.ts`
- `web/src/hooks/`
- `engine/include/pipeline/`
- `engine/include/algo/`

## Review Rule

Every finding, rule, and proposed cleanup should be checked against source files in this repository. Be especially careful with false positives around trust boundaries: internal metadata files, signed algorithm packages, MQTT messages, HTTP requests, and generated FlatBuffers code have different validation requirements.
