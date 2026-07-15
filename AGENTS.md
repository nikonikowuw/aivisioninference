# AGENTS.md — AIVisionInference

## Project Overview

**AIVisionInference** 是面向边缘设备与视频流场景的 AI 视觉推理平台。由 Go 控制面、React 管理端、C++ 推理引擎、ZLMediaKit 流媒体服务、算法包 ABI、MQTT 控制通道与 FlatBuffers 事件载荷组成，涵盖设备接入、算法包管理、推理任务编排、智能记录沉淀与系统运维。

## Tech Stack

| 层         | 技术                                                             |
| ---------- | ---------------------------------------------------------------- |
| 控制面后端 | Go 1.26+, Gin, GORM, PostgreSQL 16+, Redis 7+, Asynq, JWT, Wire  |
| 推理引擎   | C++17, CMake, FlatBuffers, FFmpeg/OpenCV, RKNN/Ascend/CUDA/Metal |
| 管理端前端 | React 19, TypeScript, Vite 6, Chakra UI 2, i18next               |
| 流媒体     | ZLMediaKit (GB28181/RTSP)                                        |
| Engine 通信 | MQTT 命令/响应、HTTP 节点心跳、FlatBuffers 事件载荷              |
| 算法包格式 | C ABI 动态库 (.so/.dylib) + 元数据 YAML                          |

## Project Structure

```text
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

## AI Coding Behavior Rules

当生成或修改代码时：

1. 优先设计数据访问方案，再编写代码。
2. 如果发现循环中需要查询数据库，必须重构为批量查询。
3. 如果无法避免多次查询，必须说明原因。
4. 不允许生成未经优化的 CRUD。
5. 所有列表查询默认按照生产环境大数据量考虑。
6. 默认假设：
   - Device 数量 > 10000
   - Algorithm 数量 > 1000
   - Event 数据 > 百万级

## 注释

### 通用原则 (General Principles)

所有语言统一遵循：

- 注释的目的是解释**"为什么这样做"**，而非"做了什么"——代码本身应能表达"做了什么"。
- 严禁保留废弃的大段注释代码、调试日志或死代码块。
- 所有遗留调试逻辑、临时的 `println`/`console.log` 等测试代码在提测前必须移除。
- 不允许对显而易见的代码添加冗余注释（如 `i++` 不需要注明"自增"）。
- `TODO` 必须附带负责人、原因和计划版本/日期，禁止无期限的 TODO。

### Go 注释规范 (Go Comment Conventions)

遵循 Go 官方 Godoc 格式，注释即文档。

- **包注释 (Package Comments)**: 每个包必须有包级别注释，说明包的职责与核心类型。格式 `// Package <name> 负责...`，放在 `package` 声明之前。
- **导出标识符注释 (Exported Identifiers)**: 所有 exported 类型、函数、常量、变量必须有 Godoc 风格注释。首句为摘要，godoc 自动提取。
- **接口注释 (Interface Comments)**: 接口定义必须注释契约语义，说明实现方需满足的行为约束。
- **函数/方法注释 (Function Comments)**: 公开函数以函数名开头，说明功能、形参含义、返回值及各错误场景。格式：`// FuncName 执行XXX操作。` 或 `// FuncName does XXX.`。
- **Handler Swagger 注解**: API Handler 必须包含 Swagger 注解（`@Summary`、`@Description`、`@Param`、`@Success`、`@Failure`、`@Router`），同时附带简短职责说明。
- **DTO/Model 字段注释**: struct 字段使用行尾注释说明含义、单位与约束（`json`/`gorm` tag 不能替代注释）。
- **复杂逻辑注释**: 涉及并发、锁、事务编排、非标准算法时必须写明设计理由与选择依据。
- **Wire Provider 注释**: Provider 函数以 `// ProvideXXX` 前缀注释，说明依赖来源与生命周期。
- **禁止 Doxygen/JSDoc 风格**: Go 中不使用 `@param`、`@return` 等结构化标签，仅用纯文本 Godoc。

### TypeScript / React 注释规范 (Frontend Comment Conventions)

前端代码（`web/`）使用 TSDoc / JSDoc 风格。

- **组件注释 (Component Comments)**: 页面级和复杂容器组件必须在文件头或组件定义前注释其职责、数据流和关键状态。
- **Props/State 接口注释**: 组件 Props 与 State 类型须包含字段说明，复杂字段需注明默认值、可选行为与依赖关系。
- **自定义 Hook 注释**: Hooks 必须注释返回值结构、内部副作用（订阅、定时器、WebSocket 绑定）和清理逻辑。
- **API 函数注释**: `services/` 中的 API 函数必须注释后端端点、请求参数含义、响应结构及错误处理策略。
- **复杂状态逻辑注释**: 涉及 `useReducer`、多状态派生、多个 `useEffect` 联动时必须写清设计理由与状态流转。
- **i18n 上下文注释**: 当 i18n key 不足以表达展示位置时，在 `t()` 调用旁追加行注释说明上下文。
- **性能优化注释**: `useMemo`、`useCallback`、`React.memo` 使用处需注明优化目标（避免何种重复计算或渲染）。
- **禁止**: 禁止在 JSX 中夹带大段注释；禁止注释无副作用的依赖数组；禁止 TSDoc `@param` 中附带非类型信息。

### C++ 注释规范 (C++ Comment Conventions)

算法包内的 C++ 源码**必须包含规范、清晰的注释**。要求如下：

- **文件头部注释 (File Headers)**: 核心接口、类定义、管线文件必须包含文件级说明，注明文件职责、逻辑结构与技术路线。
- **函数/方法注释 (Function Comments)**: 公开 API 及核心业务函数使用 Doxygen 风格，说明各形参 (`@param`)、返回值 (`@return`)、边界状态及行为规范。
- **关键算法注释 (Algorithm Step Comments)**: 非标准几何变换、自定义图像变换或后处理过滤必须逐步写明数学公式、设计原理与逻辑步骤。
- **单行微注释 (Inline Comments)**: 空指针校验、防除零保护、越界校正及资源预分配等核心步骤必须旁侧或行前追加逻辑说明。
- **保持代码整洁 (Clean Code)**: 严禁保留废弃的大段注释代码或无意义的 TODO，所有调试逻辑应当移去。
- **避免多余注释 (Avoid Redundant Comments)**: 禁止对显而易见的标准操作添加注释（如 `x++`）。

## Architecture

Go 控制面当前通过 **MQTT 命令/响应** 与 C++ 推理引擎通信，边缘节点心跳通过 **HTTP** 上报，推理事件 payload 使用 **FlatBuffers**。前端通过 **HTTP / WebSocket** 与 Go 后端交互。

```text
React 管理端 ──HTTP/WS──▶ Go 控制面 ◀─MQTT/HTTP/FlatBuffers─▶ C++ 推理引擎
                              │                          │
                              ▼                          ▼
                         PostgreSQL/Redis           RTSP / HAL / NPU
```

Go 后端分层（单向依赖）：

```text
Handler → Service → Repository → Model (GORM)
   │         │           │
   ▼         ▼           ▼
  DTO    MQTT/Task    Storage (local/pg/oss)
```

C++ 推理引擎 Pipeline：

```text
MQTT Control Plane ──→ Command Dispatcher ──→ Pipeline Pool ──→ Decoder → Preprocess → Inference (Algo .so) → Postprocess → Result Upload
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

| 规则文件                                  | 内容                                                         |
| ----------------------------------------- | ------------------------------------------------------------ |
| `.rules/golang-pro/rule.md`               | Go 后端：Context 传播、错误处理、i18n、Wire、DB、安全、Swagger、测试、代码生成 |
| `.rules/ui-ux-pro-max/rule.md`            | 前端 UI/UX：可访问性、响应式、Chakra 表单、暗色模式、动效    |
| `.rules/frontend-design/rule.md`          | 前端视觉：主题、排版、布局、图表视频、i18n 零硬编码、自检清单 |
| `.rules/vercel-react-best-practices/rule.md` | React 性能：数据请求、路由拆分、Bundle、渲染正确性、高频交互 |
| `.rules/cpp-engine-algorithm/rule.md`     | C++ Engine：架构约定、编码规范、Pipeline 性能、FlatBuffers 协议、算法包规范、C ABI 契约、构建验证、安全 |
| `.rules/development/rule.md`              | **架构规范 + 性能规范 + 禁止 N+1 查询**, 数据库操作规范      |
