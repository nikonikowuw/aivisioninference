# Pipeline And HAL

## Pipeline Ownership

Pipeline lifecycle is owned by `PipelineManager`. It creates/destroys pipelines by device/stream id, manages playback and inference enablement, owns HAL managers, and can start FFmpeg fallback paths.

Reference files:
- `engine/include/pipeline/pipeline_manager.h`
- `engine/src/pipeline/pipeline_manager.cpp`
- `engine/include/pipeline/pipeline.h`
- `engine/include/pipeline/inference_stage.h`
- `engine/include/pipeline/rtsp_push_stage.h`

Do not duplicate stream lifecycle maps outside the pipeline manager unless the data belongs to a clearly separate subsystem, such as response routing or metrics snapshots.

## HAL Contract

`engine/include/pipeline/hal.h` defines the platform media pipeline interface. HAL implementations must provide stream start/stop, pause/resume, frame callbacks, state callbacks, decode hardware type, optional encode support, capabilities, and last status.

Platform HALs are dynamic libraries. The main engine loads them through `HALManager::LoadPipeline(so_path, config_json)`. Keep platform-specific code behind the HAL interface; common pipeline code should not branch deeply on Rockchip, Ascend, macOS, or fallback implementation details.

## Buffers And Backpressure

Pipeline hot paths should minimize copies. Hardware buffers, DMA fds, encoded packets, and inference inputs should move through typed structures rather than ad hoc byte slices.

Reference files:
- `engine/include/pipeline/hw_buffer.h`
- `engine/include/pipeline/ring_queue.h`
- `engine/include/pipeline/worker_pool.h`
- `engine/include/monitor/metrics_reporter.h`

Queue capacity, eviction, frame timestamps, inference latency, and worker-idle metrics are part of operational behavior. When changing queues or workers, update metrics and tests that depend on queue depth, eviction count, and latency snapshots.

## Fallback Paths

FFmpeg/OpenCV fallback is a first-class development/runtime path, not a throwaway stub. `engine/CMakeLists.txt` requires `libavformat`, `libavcodec`, `libavutil`, `libswscale`, and `opencv4` for fallback decoding.

Changes that improve a hardware HAL should still preserve fallback behavior unless the target command explicitly narrows scope.

## Stop And Cleanup

Stopping a stream must release:

- Decoder and encoder resources
- Algorithm instances or task bindings
- Hardware buffers and frame queues
- Worker/response-router references
- FFmpeg fallback processes or decoder objects
- MQTT/control-plane state associated with the stream

Prefer idempotent `Stop()` / `Destroy()` behavior so cleanup is safe during error recovery and process shutdown.

## Tests

Use `engine/tests/` for C++ coverage. Existing test targets cover downloader behavior, device probes, snapshot JSON, heartbeat reporter, and RKMPP pipeline behavior. Add focused tests for queue semantics, HAL status mapping, command dispatch behavior, or cleanup paths when those change.
