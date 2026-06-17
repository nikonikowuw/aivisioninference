# AGENTS.md — AIVisionInference

## Project Overview

**AIVisionInference** 是面向边缘设备与视频流场景的 AI 视觉推理平台。由 Go 控制面、React 管理端、C++ 推理引擎、ZLMediaKit 流媒体服务、算法包 ABI 与 FlatBuffers IPC 协议组成，涵盖设备接入、算法包管理、推理任务编排、智能记录沉淀与系统运维。

## Tech Stack

| 层         | 技术                                                             |
| ---------- | ---------------------------------------------------------------- |
| 控制面后端 | Go 1.26+, Gin, GORM, PostgreSQL 16+, Redis 7+, Asynq, JWT, Wire  |
| 推理引擎   | C++17, CMake, FlatBuffers, FFmpeg/OpenCV, RKNN/Ascend/CUDA/Metal |
| 管理端前端 | React 19, TypeScript, Vite 6, Chakra UI 2, i18next               |
| 流媒体     | ZLMediaKit (GB28181/RTSP)                                        |
| IPC 协议   | FlatBuffers (Unix Domain Socket / TCP)                           |
| 算法包格式 | C ABI 动态库 (.so/.dylib) + 元数据 YAML                          |

## Project Structure

```
app/              # Go 控制面（cmd/internal/pkg）
engine/           # C++ 推理引擎（include/src/cmake）
algorithms/       # 算法包仓库（RKNN/Ascend/GPU/M）
web/              # React 管理端
zlm/              # ZLMediaKit 配置
proto/            # Protobuf / FlatBuffers Schema
deploy/           # Dockerfile 与部署编排
docs/             # 产品文档
openspec/         # OpenSpec 变更提案
prd/              # PRD 文档
```

## Architecture

Go 控制面通过 **FlatBuffers IPC** 与 C++ 推理引擎通信，前端通过 **HTTP / WebSocket** 与 Go 后端交互。

```
React 管理端 ──HTTP/WS──▶ Go 控制面 ──FlatBuffers──▶ C++ 推理引擎
                              │                          │
                              ▼                          ▼
                         PostgreSQL/Redis           RTSP / HAL / NPU
```

Go 后端分层（单向依赖）：

```
Handler → Service → Repository → Model (GORM)
   │         │           │
   ▼         ▼           ▼
  DTO    IPC/Task    Storage (local/pg/oss)
```

C++ 推理引擎 Pipeline：

```
IPC Server ──→ Task Manager ──→ Pipeline Pool ──→ Decoder → Preprocess → Inference (Algo .so) → Postprocess → Result Upload
```

## Dependency Injection (Google Wire)

项目使用 Google Wire 编译时依赖注入，分两层：

| 层级       | 包                 | 入口                                                  |
| ---------- | ------------------ | ----------------------------------------------------- |
| 应用层     | `internal/server/` | `InitializeApp()` — DB、Redis、JWT、WS、Router、Asynq |
| 路由业务层 | `internal/router/` | `InitializeRouteDeps()` — Repo、Service、Handler      |

- Provider 在对应 `deps.go` 中定义，在 `wire.go` 中注册
- **禁止**手动修改 `wire_gen.go`，统一通过 `make wire` 重新生成
- **禁止**绕过 Wire 手动 `new` 依赖对象

## Development Rules

项目具体开发规范按技术领域拆分在 `.rules/` 目录中，开发前必须阅读对应文件：

| 规则文件                                | 内容                                                                                                    |
| --------------------------------------- | ------------------------------------------------------------------------------------------------------- |
| `.rules/golang-pro.md`                  | Go 后端：Context 传播、错误处理、i18n、Wire、DB、安全、Swagger、测试、代码生成                          |
| `.rules/ui-ux-pro-max.md`               | 前端 UI/UX：可访问性、响应式、Chakra 表单、暗色模式、动效                                               |
| `.rules/frontend-design.md`             | 前端视觉：主题、排版、布局、图表视频、i18n 零硬编码、自检清单                                           |
| `.rules/vercel-react-best-practices.md` | React 性能：数据请求、路由拆分、Bundle、渲染正确性、高频交互                                            |
| `.rules/cpp-engine-algorithm.md`        | C++ Engine：架构约定、编码规范、Pipeline 性能、FlatBuffers 协议、算法包规范、C ABI 契约、构建验证、安全 |

<!-- TRELLIS:START -->

## Trellis Context

Managed by Trellis. Knowledge base in `.trellis/`:

- `workflow.md`: Development phases.
- `spec/`: Coding guidelines.
- `tasks/`: Active/Archived tasks.
<!-- TRELLIS:END -->
