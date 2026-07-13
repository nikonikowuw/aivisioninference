# 计划任务框架 — 执行计划

## 执行顺序概览

```
Phase 1: 基础组件
  Step 1    → EdgeNodeTag 模型 + Repository + Service + Handler
  Step 2    → 路由注册 + Wire DI

Phase 2: 计划任务框架
  Step 3    → EdgeScheduledTask 模型 + Repository
  Step 4    → EdgeScheduledTaskRecord 模型 + Repository
  Step 5    → EdgeScheduledTaskService (CRUD + MQTT 派发)
  Step 6    → Handler (CRUD + toggle + records + retry)
  Step 7    → 路由注册 + Wire DI

Phase 3: Asynq 调度
  Step 8    → EdgeScheduledTaskPatrol Asynq task
  Step 9    → EdgeScheduledTaskRecordCleanup Asynq task
  Step 10   → 注册到 Asynq mux + scheduler

Phase 4: WebSocket & MVP 验证
  Step 11   → WebSocket 事件广播
  Step 12   → 验证构建和测试
```

## 验证命令

```bash
# 每次 wire 变更后
cd app && make wire

# 每次 swagger 注解变更后
cd app && make swag

# 每次修改后验证
cd app && make build        # 编译检查
cd app && make unit-test    # 单元测试
```

## 实施步骤

### Step 1: EdgeNodeTag 模型 + Repository + Service + Handler

**文件清单（新建）：**
- `app/internal/model/edge_node_tag.go`
- `app/internal/repository/edge_node_tag.go`
- `app/internal/service/edge_node_tag.go`
- `app/internal/handler/edge_node_tag.go`
- `app/internal/dto/edge_node_tag.go`

**文件清单（修改）：**
- `app/internal/model/edge_node.go` — 添加 `Tags []EdgeNodeTag` 关联字段
- `app/internal/server/server.go` — 确认 AutoMigrate 注册新模型

**验证：** `cd app && make build`

### Step 2: 标签路由注册 + Wire DI

**文件清单（修改）：**
- `app/internal/router/deps.go` — provider 函数
- `app/internal/router/wire.go` — Wire 注册
- `app/internal/router/router.go` — 路由注册
- `app/internal/router/wire_gen.go` — `make wire` 生成

**验证：** `cd app && make wire && make build`

### Step 3: EdgeScheduledTask 模型 + Repository

**文件清单（新建）：**
- `app/internal/model/edge_scheduled_task.go`

**文件清单（修改）：**
- `app/internal/model/task_types.go` 或类似位置 — 定义命令名称常量

**验证：** `cd app && make build`

### Step 4: EdgeScheduledTaskRecord 模型 + Repository

**文件清单（新建）：**
- `app/internal/model/edge_scheduled_task_record.go`
- `app/internal/repository/edge_scheduled_task.go`
- `app/internal/repository/edge_scheduled_task_record.go`

**验证：** `cd app && make build`

### Step 5: EdgeScheduledTaskService

**文件清单（新建）：**
- `app/internal/service/edge_scheduled_task.go`

**核心方法：**
- CRUD 透传
- `DispatchCommand(ctx, nodeID, cmdName, params)` → MQTT publish
- `ExecuteTask(task)` → 解析目标节点 → 分发
- `RetryRecord(recordID)` → 重试失败的记录
- `CleanupRecords()` → 清理过期记录
- `ToggleEnabled(taskID)`

**验证：** `cd app && make build`

### Step 6: Handler

**文件清单（新建）：**
- `app/internal/handler/edge_scheduled_task.go`
- `app/internal/dto/edge_scheduled_task.go`

**验证：** `cd app && make build`

### Step 7: 计划任务路由注册 + Wire DI

**文件清单（修改）：**
- `app/internal/router/deps.go`
- `app/internal/router/wire.go`
- `app/internal/router/router.go`
- `app/internal/router/wire_gen.go` — `make wire` 生成

**验证：** `cd app && make wire && make build`

### Step 8: EdgeScheduledTaskPatrol Asynq task

**文件清单（新建）：**
- `app/internal/task/edge_scheduled_task.go`

**模式：**
- 注册为 `* * * * *` 每分钟 cron
- 读取所有 enabled 任务的 cron 表达式
- 使用 `robfig/cron` 解析并匹配当前时间
- 对匹配的任务解析目标节点 → 下发命令 → 记录结果
- 只下发给 online/enabled 节点

**验证：** `cd app && make build`

### Step 9: EdgeScheduledTaskRecordCleanup Asynq task

**文件清单（可选在 Step 8 同一文件或新建）：**
- `app/internal/task/edge_scheduled_task.go` 内新增

**模式：**
- 注册为 `0 3 * * *` 每天凌晨 3 点
- 清理 90 天前记录 + 超量记录裁剪

**验证：** `cd app && make build`

### Step 10: 注册到 Asynq mux + scheduler

**文件清单（修改）：**
- `app/internal/task/server.go` — `RegisterPeriodicTasks()` 新增
- `app/internal/server/server.go` — 确保 mux handler 注册
- `app/internal/router/router.go` — `NewAsynqMux` 注入 patrol handler

**验证：** `cd app && make build`

### Step 11: WebSocket 事件广播

在 `EdgeScheduledTaskService.DispatchCommand` 和异步完成回调中加入 `hub.Broadcast`。

**事件类型：**
- `scheduled_task.executed` — 命令下发事件
- `scheduled_task.completed` — 命令完成事件

**验证：** `cd app && make build`

### Step 12: 验证构建和测试

```bash
cd app && make wire
cd app && make build
cd app && make unit-test
```

## 风险点 / 回滚点

| 步骤 | 风险 | 回滚操作 |
|------|------|---------|
| Step 1-2 | 标签模型设计不当影响后续 | DROP TABLE 回退 |
| Step 5 | MQTT 派发逻辑需兼容 trace_id 模式 | 退回到直接调用 EngineClient |
| Step 8 | cron 表达式匹配精度 | 降级到每分钟全量执行 |
| Step 10 | Asynq 注册冲突 | 移除新增的注册行 |

## 前置检查清单（task.py start 前）

- [ ] PRD 已确认并通过用户 review
- [ ] Design 文档已完成
- [ ] 以上实施步骤的依赖关系已理清
- [ ] 新增文件的明确命名已确定
