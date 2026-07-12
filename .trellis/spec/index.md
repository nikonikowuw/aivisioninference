# AIVisionInference Trellis Specs

Project-specific coding guidance for AIVisionInference. Use these specs before editing code in the corresponding layer.

## Spec Areas

| Area | Scope |
|------|-------|
| [Backend](./backend/index.md) | Go control plane under `app/`: Gin handlers, services, repositories, GORM models, Wire, Asynq, MQTT adapters, storage, API errors |
| [Frontend](./frontend/index.md) | React/Vite admin UI under `web/`: Chakra UI, route generation, services, hooks, i18n, media components |
| [Engine](./engine/index.md) | C++ data plane under `engine/`, algorithm packages under `algorithms/`, and FlatBuffers schema under `proto/flatbuf/` |
| [Guides](./guides/index.md) | Cross-layer and code-reuse thinking guides for changes spanning multiple layers |

## Repository Boundary

AIVisionInference is a multi-runtime edge vision platform:

```text
React admin web/ -> Go control plane app/ -> MQTT/HTTP/FlatBuffers -> C++ engine/
                                         |
                                         v
                              PostgreSQL / Redis / ZLMediaKit
```

Before locating code, use CodeGraph when `.codegraph/` exists at the repository root. Fall back to `rg` and direct source reads after the graph gives the relevant packages or symbols.
