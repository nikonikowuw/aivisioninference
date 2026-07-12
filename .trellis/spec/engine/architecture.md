# Engine Architecture

## Runtime Boundary

`engine/` is a C++17 executable (`aivision-engine`) that performs data-plane work. Go owns business orchestration and persistence; C++ owns stream processing, algorithm execution, metrics, and data-plane responses.

Reference files:
- `engine/src/main.cpp` loads `.env`, environment variables, CLI overrides, HAL paths, MQTT settings, and signal handling.
- `engine/src/engine.cpp` wires pipeline, worker pool, algorithm manager, response router, heartbeat, metrics, and MQTT control.
- `engine/CMakeLists.txt` defines source lists, dependencies, HAL options, tests, and generated-header installation.

## Module Layout

- `engine/include/`: public/internal headers. New modules need headers here.
- `engine/src/`: implementations. Keep source files parallel to header subdirectories.
- `engine/src/pipeline/` and `engine/include/pipeline/`: pipeline, stages, worker pool, queues, HAL manager, fallback decoder.
- `engine/src/algo/` and `engine/include/algo/`: dynamic library loading, algorithm instance lifecycle, ABI contract, downloader.
- `engine/src/monitor/` and `engine/include/monitor/`: heartbeat reporter, metrics reporter, device monitor, hardware probes.
- `engine/include/proto/flatbuf/`: generated C++ FlatBuffers headers.
- `engine/tests/`: C++ tests for downloader, monitor, device probes, heartbeat, pipeline, snapshot JSON, and platform behavior.

Add new engine source files to `ENGINE_LIB_SOURCES`, `ENGINE_SOURCES`, headers, and tests in `engine/CMakeLists.txt`.

## Configuration Flow

`engine/src/main.cpp` documents the precedence: defaults, `.env`, environment variables, then CLI arguments. Environment variables use the `NIKO_ENGINE_` prefix, such as:

- `NIKO_ENGINE_WORKERS`
- `NIKO_ENGINE_HAL_SO`
- `NIKO_ENGINE_FALLBACK_HAL_SO`
- `NIKO_ENGINE_HAL_PLATFORM`
- `NIKO_ENGINE_ENABLE_MQTT`
- `NIKO_ENGINE_MQTT_BROKER`
- `NIKO_ENGINE_NODE_ID`
- `NIKO_ENGINE_AUTH_TOKEN`
- `NIKO_ENGINE_ALGO_DIR`

Do not add unprefixed engine environment variables.

## Threading And Lifecycle

Use RAII and explicit `Start()` / `Stop()` lifecycle methods. Threaded modules should use atomics, mutexes, condition variables, or `asio::io_context` with clear shutdown behavior.

Reference files:
- `engine/include/mqtt_control_plane.h` owns an `asio::io_context`, work guard, and worker threads for command offloading.
- `engine/include/monitor/metrics_reporter.h` owns an atomic running flag, reporter thread, mutex, condition variable, and callback.

Engine code must isolate errors to a stream, task, algorithm, or command where possible. A malformed MQTT message, bad algorithm package, or one failed stream should not crash the process.

## Logging And Hot Paths

Avoid high-volume `std::cout` / `std::cerr` in frame-level paths. Startup configuration printing in `main.cpp` is acceptable; per-frame logging in worker queues, decode callbacks, or inference stages is not.

When adding logs, include task/stream/algo identifiers and status codes. Do not log auth tokens, MQTT passwords, presigned download URLs with credentials, or full biometric payloads.
