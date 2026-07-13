# Remote Ops: Scheduled Tasks & Web Terminal

## Goal

为边缘节点提供远程运维能力：支持对边缘节点执行计划任务（定时操作）和提供 Web 终端访问（实时交互式操作）。

## Structure

此任务为父任务，包含两个独立交付的子任务：

| 子任务 | 职责 | 交付物 |
|--------|------|--------|
| **edge-scheduled-tasks** | 计划任务管理：定时向边缘节点下发运维命令 | CRUD + Asynq cron + MQTT 命令派发 |
| **edge-web-terminal** | Web 终端：浏览器端实时远程访问边缘节点 shell | WebSocket 代理 + 会话管理 + 终端 UI 配合 |

两个子任务可独立规划、实施、检查和归档。

## Confirmed Facts

### 现有基础设施

- **Asynq 定时任务系统**：已使用 `github.com/hibiken/asynq`，通过 `asynq.Scheduler` 管理定时任务，支持 cron 表达式。现有定时任务：
  - `*/5 * * * *` — 边缘节点状态检查 (`TypeDeviceStatusCheck`)
  - `* * * * *` — AI 推理任务巡检 (`TypeAIVisionTaskPatrol`)
  - 存储清理通过 `StorageScheduler` 动态管理
  - `*/1 * * * *` — 计划任务巡检 (`TypeEdgeScheduledTaskPatrol`)
  - `0 3 * * *` — 计划任务清理 (`TypeEdgeScheduledTaskCleanup`)
  - `*/5 * * * *` — 终端会话清理 (`TypeTerminalSessionCleanup`)
- **MQTT 命令/响应模式**：引擎通信通过 MQTT，命令 Topic 模式为 `aivision/edge/{nodeID}/cmd/{command}`，通过 `trace_id` + Redis Pub/Sub 实现同步等待。现有的 EngineClient 接口包含：StartStream, StopStream, StartPlayback, StopPlayback, GetStreamStatus, StartSelfCheck, WarmupAlgorithm, UpdateFaceLibrary, ExtractFaceEmbedding
- **AI 时间配置**：`model.AITimeSchedule` 提供日期范围 + 每日时间窗口（JSONB）配置，已有完整 CRUD，被 `AIVisionTask` 引用
- **边缘节点模型**：`model.EdgeNode` 包含状态、能力字段、心跳跟踪、引擎版本信息，含 SSH 端口 (`SSHPort`) 和加密存储的 SSH 私钥 (`SSHPrivateKey`)
- **任务挂起原因**：`node_offline`, `manual`, `schedule`, `error` — 用于任务暂停/恢复控制
- **WebSocket Hub**：已实现 WebSocket Hub (`ws.Hub`) 用于向管理端推送实时事件
- **通用远程命令框架**：已实现 `EdgeScheduledTask` 模型 + `EdgeScheduledTaskService`（DB 驱动定时巡检 + MQTT 直接派发），支持 cron 表达式、节点/标签目标、JSON 参数、可选同步等待重试
- **SSH/终端能力**：已实现完整的 Web 终端系统：SSH 连接池 (`SSHPool`)、会话生命周期管理 (`TerminalSessionService`)、ttyrec 录制器 (`Recorder`)、WebSocket 代理终端 (`TerminalHandler`)、会话清理定时任务

## Implementation Summary

### edge-scheduled-tasks (已完成)

| 文件 | 职责 |
|------|------|
| `app/internal/model/edge_scheduled_task.go` | `EdgeScheduledTask` 数据模型，9个命令常量与 C++ CommandDispatcher 对齐 |
| `app/internal/model/edge_scheduled_task_record.go` | `EdgeScheduledTaskRecord` 执行记录模型 |
| `app/internal/repository/edge_scheduled_task.go` | GORM CRUD 查询 |
| `app/internal/repository/edge_scheduled_task_record.go` | 执行记录查询 |
| `app/internal/service/edge_scheduled_task.go` | 定时调度、目标解析、MQTT 命令派发、重试、WS 事件通知 |
| `app/internal/handler/edge_scheduled_task.go` | CRUD + toggle + retry HTTP API + Swagger 注解 |
| `app/internal/task/edge_scheduled_task.go` | Asynq 定时巡逻（每60s）+ 每日清理（凌晨3点） |
| `app/internal/router/edge_node.go` | 路由注册 |

API 端点：`/api/v1/edge-scheduled-tasks/*` — CRUD + toggle + list records + retry record

### edge-web-terminal (已完成)

| 文件 | 职责 |
|------|------|
| `app/internal/model/terminal_session.go` | `TerminalSession` 数据模型 (active/paused/closed) |
| `app/internal/repository/terminal_session.go` | GORM CRUD 查询 |
| `app/internal/service/terminal_session.go` | 会话生命周期：创建、恢复、暂停、关闭 |
| `app/internal/service/ssh_pool.go` | SSH 连接池（引用计数，5分钟空闲超时） |
| `app/internal/service/recorder.go` | ttyrec I/O 录制器 |
| `app/internal/handler/terminal.go` | WebSocket SSH 桥接 + terminalMessage 协议 |
| `app/internal/task/terminal_session.go` | 超时会话清理（每5分钟） |
| `app/internal/router/edge_node.go` | 路由注册
| `web/src/services/terminal.ts` | 前端 API 客户端 |
| `web/src/hooks/useTerminal.ts` | React WebSocket + xterm.js 钩子 |
| `web/src/components/terminal/TerminalView.tsx` | 终端 UI 组件 |
| `web/src/components/terminal/PlaybackView.tsx` | ttyrec 回放组件 |
| `web/src/views/.../TerminalTab.tsx` | 节点详情页终端标签 |
| `web/src/views/.../playback.tsx` | 录制回放页面 |

API 端点：
- `GET /api/v1/ws/terminal` — WebSocket 终端
- `GET /api/v1/edge-nodes/:id/sessions` — 列会话
- `GET /api/v1/edge-nodes/:id/sessions/:sid` — 查会话
- `DELETE /api/v1/edge-nodes/:id/sessions/:sid` — 关闭会话
- `GET /api/v1/edge-nodes/:id/sessions/:sid/recording` — 取录制

## Acceptance Criteria

- [x] 两个子任务分别有独立的交付物和验收标准（通过父任务 PRD 管理，每个子任务有独立的 handler/model/service 实现）
- [x] 子任务间无循环依赖或隐含阻塞关系（scheduled-tasks 和 web-terminal 使用独立基础设施，不互相依赖）

## Out of Scope

- 此处只定义父任务范围和子任务划分，详细实现见各代码模块
