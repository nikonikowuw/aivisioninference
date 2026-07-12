# C++ Engine And Protocol Specs

These specs cover `engine/`, `algorithms/`, and `proto/flatbuf/`. The engine is the data plane: video stream lifecycle, HAL integration, algorithm dynamic loading, inference workers, MQTT control handling, heartbeat/metrics reporting, and FlatBuffers event payloads.

## Guides

| Guide | Use When |
|-------|----------|
| [Architecture](./architecture.md) | Adding engine modules, build targets, device monitoring, MQTT handlers, or lifecycle code |
| [Pipeline And HAL](./pipeline-hal.md) | Changing video decode/playback/inference pipelines, hardware buffers, worker queues, or platform HALs |
| [Protocols And Algorithms](./protocols-algorithms.md) | Changing MQTT command/response payloads, FlatBuffers schema, algorithm package metadata, or C ABI |

## Local Anchors

- Engine entrypoint: `engine/src/main.cpp`
- Core engine orchestration: `engine/src/engine.cpp`, `engine/include/engine.h`
- MQTT control plane: `engine/src/mqtt_control_plane.cpp`, `engine/include/mqtt_control_plane.h`
- Command dispatch: `engine/src/command_dispatcher.cpp`
- Response routing: `engine/src/response_router.cpp`, `engine/include/response_router.h`
- Pipeline: `engine/src/pipeline/`, `engine/include/pipeline/`
- Algorithm loading: `engine/src/algo/`, `engine/include/algo/`
- Device/metrics monitoring: `engine/src/monitor/`, `engine/include/monitor/`
- Protocol schema: `proto/flatbuf/`
- Algorithm packages: `algorithms/<name>/<version>/`

## Required Checks

- Engine build/test: `cd engine && make test`
- Engine build only: `cd engine && make dev` or the target-specific Makefile command
- FlatBuffers changes: regenerate C++ and Go code, then build both `app/` and `engine/`
