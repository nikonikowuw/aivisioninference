# AIVisionInference

<p align="center">
  <a href="./README.md">简体中文</a> | <strong>English</strong>
</p>

<p align="center">
  <a href="https://go.dev/"><img src="https://img.shields.io/badge/Go-1.23+-00ADD8?logo=go&logoColor=white" alt="Go Version"></a>
  <a href="https://gin-gonic.com/"><img src="https://img.shields.io/badge/Gin-v1.10-blue?logo=go" alt="Gin"></a>
  <a href="https://gorm.io/"><img src="https://img.shields.io/badge/GORM-v1.26-5c6bc0?logo=go" alt="GORM"></a>
  <a href="https://www.postgresql.org/"><img src="https://img.shields.io/badge/PostgreSQL-16-4169E1?logo=postgresql&logoColor=white" alt="PostgreSQL"></a>
  <a href="https://redis.io/"><img src="https://img.shields.io/badge/Redis-7-DC382D?logo=redis&logoColor=white" alt="Redis"></a>
  <a href="https://isocpp.org/"><img src="https://img.shields.io/badge/C++-17-00599C?logo=cplusplus&logoColor=white" alt="C++17"></a>
  <a href="https://mqtt.org/"><img src="https://img.shields.io/badge/MQTT-Control%20Plane-660066" alt="MQTT"></a>
  <a href="https://flatbuffers.dev/"><img src="https://img.shields.io/badge/FlatBuffers-Event%20Payload-orange" alt="FlatBuffers"></a>
  <a href="https://vite.dev/"><img src="https://img.shields.io/badge/Vite-6.x-646CFF?logo=vite&logoColor=white" alt="Vite"></a>
  <a href="https://chakra-ui.com/"><img src="https://img.shields.io/badge/Chakra--UI-2.x-319795?logo=chakra-ui&logoColor=white" alt="Chakra UI"></a>
</p>

AIVisionInference is an **AI vision inference platform** for edge devices and video-stream workloads. The system combines a Go control plane, a React admin console, a C++ inference data plane, ZLMediaKit, algorithm package ABI, an MQTT control channel, and FlatBuffers event payloads for device access, algorithm management, task orchestration, smart-record persistence, and operations.

---

## Contents

- [Core Capabilities](#core-capabilities)
- [Architecture](#architecture)
- [Tech Stack](#tech-stack)
- [Quick Start](#quick-start)
- [Project Layout](#project-layout)
- [Common Commands](#common-commands)
- [Algorithm Package Layout](#algorithm-package-layout)
- [Engine Communication](#engine-communication)
- [Configuration](#configuration)
- [Related Docs](#related-docs)

---

## Core Capabilities

- Video stream and device management for RTSP and GB28181 workloads, backed by ZLMediaKit.
- AI task orchestration from the Go control plane to the C++ inference engine.
- Algorithm package management with dynamically loaded shared libraries and metadata.
- Smart-record persistence for inference results, alarms, snapshots, and task state.
- Edge-node management with health monitoring, package deployment, and version compatibility checks.
- Reused admin-platform capabilities including JWT auth, RBAC, audit logs, file handling, i18n, and Swagger.
- Hardware-adaptation hooks for RKMPP, RGA, FFmpeg/OpenCV fallback, and future accelerators.

---

## Architecture

```txt
┌──────────────────────┐
│ React Admin web/      │
└──────────┬───────────┘
           │ HTTP / WebSocket
┌──────────▼───────────┐      MQTT commands/responses      ┌──────────────────────┐
│ Go Control Plane app/ │ ◀──────────────────────────────▶ │ C++ Engine engine/    │
│ Gin + GORM + Redis    │ ───── HTTP heartbeats/deploy ──▶ │ Pipeline + Algo .so   │
└──────┬─────────┬──────┘      status / events / metrics   └──────────┬───────────┘
       │         │                                                    │
       ▼         ▼                                                    ▼
 PostgreSQL   Redis / Asynq                                      RTSP / HAL / NPU
       │
       ▼
 ZLMediaKit / File Storage / Smart Records
```

The Go backend follows a layered architecture:

```txt
Handler → Service → Repository → Model
   │         │
   ▼         ▼
  DTO   Storage / MQTT / Task
```

---

## Tech Stack

### Control Plane and Admin UI

- Backend: Go 1.23+, Gin, GORM, PostgreSQL 16, Redis 7, Asynq, MQTT, JWT, Wire, Viper, Zap, Swagger
- Frontend: React 19, TypeScript, Vite 6, Chakra UI 2, React Router, TanStack Table, i18next

### Data Plane and Protocols

- Inference engine: C++17, CMake, MQTT, FlatBuffers, FFmpeg, OpenCV, dynamic-library ABI
- Hardware adaptation: RKMPP, RGA, DMA Buffer, VideoToolbox/macOS stub, x86_64 stub fallback
- Streaming: ZLMediaKit, using `zlmediakit/zlmediakit:latest` in the root `docker-compose.yml`

---

## Quick Start

### Prerequisites

- Go 1.23+
- Node.js 18+ / npm
- Docker and Docker Compose
- CMake, a C++17 compiler, and FlatBuffers `flatc`
- Optional: FFmpeg, OpenCV, Rockchip MPP/RGA development packages

### 1. Start PostgreSQL and Redis

```bash
cd app
make docker-up-deps
```

The root `docker-compose.yml` starts `postgres` and `redis` with these defaults:

```txt
POSTGRES_USER=aivision
POSTGRES_PASSWORD=aivision123
POSTGRES_DB=aivision
```

### 2. Start the Go control plane

```bash
cd app
cp -n .env.example .env
make migrate
make serve
```

Default endpoints:

- API: `http://localhost:8080`
- WebSocket: `http://localhost:8090`
- Swagger: `http://localhost:8080/swagger/index.html`

### 3. Start the React admin console

```bash
cd web
npm install
npm run dev
```

Default frontend URL: `http://localhost:5173`

### 4. Build the C++ engine

```bash
cd engine
make dev
```

The development build disables RKMPP and uses the stub HAL. Artifacts are generated in `engine/build/`.

### 5. Run the full stack with Docker Compose

```bash
docker-compose up -d --build
```

The root `docker-compose.yml` orchestrates PostgreSQL, Redis, ZLMediaKit, the Go control plane, and the C++ engine. The repository already includes `deploy/Dockerfile.engine`, but the `go-server` service still references `deploy/Dockerfile`, which is not present in the current tree.

---

## Project Layout

```txt
AIVisionInference/
├── app/                    # Go control plane: API, RBAC, devices, tasks, records, MQTT, migrations
├── web/                    # React admin console
├── engine/                 # C++ inference data plane
├── algorithms/             # Sample algorithm packages and model assets
├── proto/flatbuf/          # Shared FlatBuffers message schema and compatibility notes
├── zlm/                    # ZLMediaKit configuration and related assets
├── deploy/                 # Deployment files, engine Dockerfile, scripts
├── docs/                   # Product and operations docs
├── openspec/               # OpenSpec proposals and specs
└── docker-compose.yml      # Local orchestration
```

---

## Common Commands

### Go control plane

```bash
cd app
make serve
make dev
make run
make build
make build-linux
make migrate
make wire
make swag
make gen
make unit-test
make lint
```

### React admin console

```bash
cd web
npm run dev
npm run build
npm run preview
npm run test
```

### C++ engine

```bash
cd engine
make dev
make release
make macos
make rk3568
make rk3568-cross SYSROOT=/path/to/sysroot
make flatbuf
make test
make clean
```

---

## Algorithm Package Layout

Algorithm packages live under `algorithms/` and typically contain:

```txt
algorithm-name/
└── version/
    ├── algo_meta.yaml
    ├── libxxx.so
    ├── models/
    ├── label_map.json
    └── examples/
```

The C++ engine loads algorithm packages as shared libraries through a unified C ABI. See:

- `algorithms/face_recognition/1.0.0/README.md`
- `algorithms/fall_detection/1.0.0/README.md`

---

## Engine Communication

The active runtime path is MQTT-first:

- Control commands: Go `EngineClient` publishes JSON commands over MQTT, and C++ `MqttControlPlane` forwards them to `CommandDispatcher`.
- Sync-over-async: payloads carry a `trace_id`; Go waits on Redis Pub/Sub for the matching response.
- Command responses: C++ handlers may still build FlatBuffers responses internally, then `ResponseRouter` adapts them into MQTT JSON responses under `aivision/edge/{node_id}/response/{cmd}`.
- Inference events: C++ publishes `InferenceResultMsg` over MQTT with a FlatBuffers envelope payload.
- Node heartbeat: C++ `HeartbeatReporter` uses HTTP `POST /api/v1/edge-nodes/{id}/heartbeat` and receives pending package deployments in the response.

The FlatBuffers schema under `proto/flatbuf/` remains relevant for event payloads, internal response structures, and shared message models reused by the current MQTT/HTTP runtime.

Key message types include:

- `StartStreamCmd` / `StopStreamCmd`
- `UpdateConfigCmd`
- `LoadAlgoCmd` / `UnloadAlgoCmd`
- `InferenceResultMsg`
- `StreamStatusMsg`
- `EngineMetricsMsg`
- `HeartbeatCmd` / `HeartbeatAckMsg` as legacy FlatBuffers heartbeat models; the active node heartbeat path is HTTP

See:

- `proto/flatbuf/README.md`
- `proto/flatbuf/CHANGELOG.md`
- `proto/flatbuf/COMPATIBILITY.md`

---

## Edge Node Management

The edge-node module registers the C++ engine as a manageable edge node and adds health monitoring, package deployment, and version compatibility checks.

### Flow summary

1. An administrator creates an edge node and receives a JWT token.
2. The engine is configured with `NIKO_ENGINE_PLATFORM_URL`, `NIKO_ENGINE_NODE_ID`, and `NIKO_ENGINE_AUTH_TOKEN`.
3. The engine reports status, hardware information, load, and installed algorithms every 5 seconds.
4. The platform returns pending deployments in the heartbeat response.
5. The engine downloads, verifies, extracts, and installs algorithm packages.

### Key properties

- JWT-based node authentication
- Presigned download URLs for algorithm packages
- Retry logic for failed deployments
- Online-node load recommendation during task assignment
- Minimum compatible engine-version checks
- WebSocket push for node-state updates

Relevant docs:

- `docs/edge-node-guide.md`
- `docs/engine-setup.md`
- `docs/api.md`

---

## Configuration

Configuration priority:

```txt
environment variables > .env > config.yaml > defaults
```

All environment variables use the `NIKO_` prefix. Secrets must be injected via environment variables in production.

Common control-plane settings:

```ini
NIKO_APP_ENV=dev
NIKO_APP_PORT=8080
NIKO_DB_HOST=localhost
NIKO_DB_PORT=5432
NIKO_DB_USER=aivision
NIKO_DB_PASSWORD=aivision123
NIKO_DB_NAME=aivision
NIKO_REDIS_HOST=localhost
NIKO_REDIS_PORT=6379
NIKO_JWT_SECRET=change-me-in-production
NIKO_MQTT_HOST=localhost
NIKO_MQTT_PORT=1883
NIKO_STORAGE_DRIVER=local
```

Common engine settings:

```ini
NIKO_ENGINE_PLATFORM_URL=http://your-platform:8080
NIKO_ENGINE_NODE_ID=<node-id>
NIKO_ENGINE_AUTH_TOKEN=<jwt-token>
NIKO_ENGINE_ENABLE_MQTT=true
NIKO_ENGINE_MQTT_BROKER=tcp://localhost:1883
NIKO_ENGINE_WORKERS=4
NIKO_ENGINE_HAL_PLATFORM=macos
NIKO_ENGINE_RTSP_PUSH=rtsp://localhost:10554
```

Security requirements:

- Change `NIKO_JWT_SECRET`, database passwords, and storage credentials in production.
- Do not commit real secrets.
- Persist upload directories, algorithm packages, and model files according to your deployment layout.

---

## Related Docs

- `engine/README.md` for engine runtime and HAL notes
- `proto/flatbuf/README.md` for FlatBuffers schema and communication notes
- `docs/gb28181-guide.md` for GB28181 integration
- `docs/zlm-verification.md` for ZLMediaKit verification
- `docs/edge-node-guide.md` for operator workflows
- `docs/engine-setup.md` for engine configuration and troubleshooting
- `docs/api.md` for API references
- Swagger at `http://localhost:8080/swagger/index.html` after the Go service starts

---

## License

Add the repository license and copyright notice according to the actual distribution policy.
