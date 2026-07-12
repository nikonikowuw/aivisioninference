# Protocols And Algorithms

## Control And Event Channels

The current runtime channel is not a pure FlatBuffers IPC system:

- Go publishes MQTT JSON commands through `app/internal/service/mqtt_engine_client.go`.
- C++ subscribes and dispatches commands in `engine/src/mqtt_control_plane.cpp` and `engine/src/command_dispatcher.cpp`.
- C++ builds command responses through `ResponseRouter` and publishes MQTT JSON responses.
- Inference events are MQTT messages whose payload is wrapped by FlatBuffers `ControlEnvelope`.
- Edge-node heartbeat is HTTP: `POST /api/v1/edge-nodes/{id}/heartbeat`.

`proto/flatbuf/README.md` is the source of truth for schema role and generation commands.

## FlatBuffers Schema Evolution

Schema files live in `proto/flatbuf/`:

- `common.fbs`
- `envelope.fbs`
- `commands.fbs`
- `results.fbs`
- `CHANGELOG.md`
- `COMPATIBILITY.md`

When changing schemas:

1. Add optional fields where possible.
2. Do not reorder existing fields.
3. Mark old fields deprecated instead of deleting them.
4. Update `CHANGELOG.md` and `COMPATIBILITY.md`.
5. Regenerate Go output under `app/internal/pkg/controlproto/fbs`.
6. Regenerate C++ output under `engine/include/proto/flatbuf`.
7. Build/test both Go and engine sides.

Do not manually edit generated FlatBuffers code.

## MQTT Command Compatibility

MQTT JSON is the command-control path, so schema changes alone are not enough for command behavior. Update the Go command builder, C++ dispatcher, response router, tests, and frontend/service expectations when command fields change.

Preserve traceability fields such as sequence id, timestamp, node id, task id, and stream id. They are needed to debug cross-process issues.

## Scenario: Preview Capacity Admission

### 1. Scope / Trigger

- Trigger: changing preview admission, Engine playback lifecycle, `EngineMetricsMsg`, or media-capacity reservations.
- The contract spans Engine environment configuration, `PipelineManager`, FlatBuffers, Go metrics caching, and database-backed admission.

### 2. Signatures

- Environment: `NIKO_ENGINE_MAX_PREVIEW_STREAMS=<positive uint32>`.
- FlatBuffers: `EngineMetricsMsg.preview_capacity:uint`, `preview_in_use:uint`, `preview_capacity_valid:bool`, plus `timestamp_ns:uint64`.
- Go admission: `MediaCapacityScheduler.Reserve(ctx, streamKey, lease)` reserves exactly one preview slot.
- Database: `media_capacity_reservations.preview_slots` is `1` for every new preview reservation; existing decode/encode/egress columns are compatibility-only and must not affect admission.

### 3. Contracts

- `preview_capacity` is a device-qualified deployment value owned by Engine configuration. Go must not derive or override it from decoder, encoder, bandwidth, NPU, or `current_load` values.
- `preview_in_use` counts unique pipelines whose playback output is enabled. Pure inference counts zero; inference plus playback counts one.
- `CreatePipeline`, `EnablePlayback`, `DisablePlayback`, and `DestroyPipeline` must keep playback state idempotent so repeated commands cannot change the count twice.
- Go admits only when the node is online/enabled, the snapshot and receive time are within TTL, the validity bit is true, and `preview_in_use + pending + 1 <= preview_capacity`.
- Eligible nodes sort by post-admission ratio `(preview_in_use + pending + 1) / preview_capacity`, then ascending node ID.

### 4. Validation & Error Matrix

| Condition | Result |
|-----------|--------|
| Environment missing, zero, malformed, trailing text, or greater than `uint32` | Log a configuration error, report `preview_capacity_valid=false`, reject new automatic preview admission |
| Old Engine omits appended fields | FlatBuffers zero defaults produce `preview_capacity_valid=false`; reject admission |
| `preview_in_use > preview_capacity` | Treat snapshot as invalid; reject admission |
| Snapshot or Go receive time exceeds TTL | Reject with `stale_metrics` |
| `preview_in_use + pending + 1 > preview_capacity` | Reject with `preview_full` |
| Repeated reservation for the same `stream_key` | Return the existing effective reservation without consuming another slot |

### 5. Good/Base/Bad Cases

- Good: capacity `8`, in use `5`, pending `1` admits one preview and scores `7/8`.
- Base: capacity `1`, in use `0`, pending `0` admits exactly one concurrent requester.
- Bad: validity false, capacity zero, stale timestamp, or total usage equal to capacity rejects the request.

### 6. Tests Required

- Protocol conversion asserts all three preview fields survive direct and enveloped FlatBuffers payloads.
- Engine tests/build assert configured capacity is reported and playback-enabled pipelines determine `preview_in_use`.
- Scheduler tests assert lowest post-admission ratio, node-ID tie break, `preview_full`, stale/invalid metrics, lease expiry, and no oversubscription of the final slot.
- Rolling-upgrade tests/assertions treat a payload without the appended fields as unschedulable.

### 7. Wrong vs Correct

Wrong:

```text
admit = decode_slots && encode_slots && egress_bandwidth
```

Correct:

```text
admit = preview_capacity_valid && fresh && preview_in_use + pending + 1 <= preview_capacity
```

## Algorithm Package Layout

Algorithm packages live under:

```text
algorithms/<algorithm_name>/<version>/
```

Expected package contents include:

- `algo_meta.yaml`
- `label_map.json`
- `models/`
- `src/`
- `CMakeLists.txt` or `build.sh`
- `test.sh`
- `README.md`

The metadata file is the control-plane parsing entrypoint. Keep algorithm id, version, domain, capabilities, result schema, parameter schema, platform support, and package integrity fields synchronized with backend validation and frontend upload/deploy UI.

## C ABI

The dynamic library contract is defined in `engine/include/algo/abi_contract.h`. Algorithm libraries must export C ABI functions with `extern "C"`, return stable error codes, keep result memory allocation/freeing paired, and catch C++ exceptions before crossing the ABI boundary.

Reference files:
- `engine/include/algo/abi_contract.h`
- `engine/include/algo/algo_instance.h`
- `engine/include/algo/algo_manager.h`
- `engine/include/algo/algorithm_downloader.h`

Do not let algorithms depend on Go process memory, Go private types, or frontend-only assumptions. Communication happens through ABI structs, JSON config, labels, and protocol payloads.

## Result JSON

Inference result JSON must be stable and versionable. Keep field names aligned with Go DTO/database mappings and frontend expectations. Fields such as `category_code`, bbox, confidence, track id, timestamp, ROI, and evidence path need clear units and coordinate systems.

If label maps, category codes, thresholds, input dimensions, or NMS settings change, update `algo_meta.yaml`, `label_map.json`, Go validation/mapping, and frontend display logic together.
