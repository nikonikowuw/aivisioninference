# 计划任务：定时向边缘节点下发运维命令

## Goal

构建一个通用的计划任务框架：支持 CRUD 管理定时计划任务，通过 Asynq cron 定时触发，通过 MQTT 向边缘节点下发命令，并提供执行记录与结果回溯能力。

## Confirmed Facts

### 现有基础设施（可复用）

- **Asynq 定时任务系统** (`app/internal/task/`)：Asynq Server + Scheduler 已部署运行，支持 cron 表达式
- **MQTT 命令/响应模式** (`app/internal/service/mqtt_engine_client.go`)：
  - 命令 Topic 模式：`aivision/edge/{nodeID}/cmd/{command}`
  - `trace_id` + Redis Pub/Sub (`MqttSyncManager`) 实现同步等待响应
  - `MqttEngineClient` 实现了所有 9 种硬编码命令（start_stream, stop_stream, self_check, algo_warmup, face_library, face_embedding, 等）
- **边缘节点模型** (`model.EdgeNode`)：节点 ID、名称、状态、能力字段
- **DB**: PostgreSQL + GORM
- **WebSocket Hub** (`ws.Hub`)：可向管理端推送任务执行事件
- **计划任务挂起原因**：`model.SuspendedReasonSchedule` 已定义（`"schedule"`），但尚未在 `ai_vision_tasks.suspended_reason` 中使用

### 范围决策

- **只做框架，不做引擎侧扩展**：命令类型通过字符串 `command_name` 标识，当前框架支持对已有 MQTT 命令的定时派发
- **框架为本任务的交付核心**：CRUD + cron 调度 + MQTT 下发 + 执行记录
- **节点选择方案**：第一期同时支持单节点和标签分组（需要新建 EdgeNode 标签系统）
- **执行模式**：默认 fire-and-forget（记录 `dispatched`），可选 `wait_response=true` 复用 trace_id 同步等待响应
- **命令定义**：Go 常量枚举（与现有 EngineClient 命令对齐），CRUD 校验只允许枚举值
- **执行记录保留**：每任务保留最近 100 条，且自动清理 90 天前的过期记录

### 执行记录状态机

```
pending → dispatched → success
                    ↘ failed
```

## Requirements

### 基础组件：EdgeNode 标签系统

- [ ] 新建 `EdgeNodeTag` model：id, name, color, description
- [ ] EdgeNode ↔ EdgeNodeTag 多对多关联（中间表 `edge_node_tags`）
- [ ] 标签 CRUD API
- [ ] 标签管理不对齐在计划任务中——独立可复用的基础功能

### 功能需求

- [ ] 计划任务 CRUD：创建/编辑/删除定时任务定义
- [ ] 定时任务定义包含：名称、描述、cron 表达式、目标（单节点 or 标签）、命令名称、命令参数（JSON）
- [ ] 节点选择支持：单个节点、标签分组
- [ ] 系统根据 cron 表达式自动调度 MQTT 命令下发
- [ ] 执行记录：每次下发产生一条记录，记录状态（pending/success/failed）+ 结果/错误信息
- [ ] 已下发的执行记录支持回溯查询和重试
- [ ] 任务支持 enable/disable 控制，disable 后不再被 cron 触发
- [ ] 通过 WebSocket 实时推送任务执行状态变更

### 非功能需求

- 复用现有 Asynq Scheduler 注册机制
- 复用现有 MQTT EngineClient 通信链路
- 定时调度精度：分钟级（与现有 Asynq cron 一致）
- 计划任务数据持久化到 PostgreSQL

## Acceptance Criteria

- [ ] 计划任务 CRUD API 通过测试
- [ ] Asynq cron 在指定时间正确下发 MQTT 命令
- [ ] 执行记录产生并可查询
- [ ] Enable/disable 控制有效
- [ ] WebSocket 推送任务执行事件
- [ ] 兼容现有测试和构建流程（`make wire`、`make build`、`make unit-test`）

## Out of Scope

- Engine 侧 C++ 新增命令类型（由后续任务处理）
- 复杂工作流/任务链编排
- 定时任务依赖管理

## Open Questions

- 已全部通过提问确认，无需额外开放问题
