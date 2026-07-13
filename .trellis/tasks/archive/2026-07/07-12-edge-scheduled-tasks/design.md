# 计划任务框架 — 技术设计方案

## 1. 架构总览

```
┌──────────────────────────────────────────────────────┐
│                  计划任务框架 (Go 控制面)               │
│                                                        │
│  ┌──────────┐  ┌──────────────┐  ┌──────────────────┐ │
│  │ CRUD API │  │ Asynq Patrol │  │ 执行记录 & 清理    │ │
│  │ (Handler)│  │ (1min cron)  │  │ (Asynq + 定时)    │ │
│  └────┬─────┘  └──────┬───────┘  └────────┬─────────┘ │
│       │               │                    │           │
│  ┌────▼───────────────▼────────────────────▼─────────┐ │
│  │           EdgeScheduledTaskService                  │ │
│  │  (CRUD + 调度解析 + MQTT 派发 + 记录管理)         │ │
│  └──────────────────────────┬────────────────────────┘ │
│                             │                          │
│  ┌──────────────────────────▼────────────────────────┐ │
│  │   MqttEngineClient / 直接 MQTT Publish             │ │
│  │   (按模式选择：fire-and-forget 或 trace_id 同步)   │ │
│  └──────────────────────────┬────────────────────────┘ │
└─────────────────────────────┼──────────────────────────┘
                              │
              aivision/edge/{nodeID}/cmd/{command}
                              │
              ┌───────────────▼────────────────┐
              │  C++ Engine CommandDispatcher    │
              │  (现有，不做修改)                │
              └──────────────────────────────────┘
```

## 2. 新增数据模型

### 2.1 EdgeNode — Tag 关联（独立基础组件）

```go
// EdgeNodeTag 边缘节点标签
type EdgeNodeTag struct {
    BaseModel
    Name        string     `gorm:"type:varchar(100);not null;uniqueIndex;comment:标签名称"`
    Color       string     `gorm:"type:varchar(7);default:'#1890ff';comment:标签颜色(Hex)"`
    Description string     `gorm:"type:varchar(500);comment:描述"`
    EdgeNodes   []EdgeNode `gorm:"many2many:edge_node_tag_relations;"`
}
```

**中间表** `edge_node_tag_relations`（GORM auto-migrate）：
- `edge_node_id` (FK → edge_nodes.id, ON DELETE CASCADE)
- `edge_node_tag_id` (FK → edge_node_tags.id, ON DELETE CASCADE)
- 复合唯一索引 `(edge_node_id, edge_node_tag_id)`

### 2.2 EdgeScheduledTask 计划任务定义

```go
type EdgeScheduledTask struct {
    BaseModel
    Name            string         `gorm:"type:varchar(255);not null;comment:任务名称"`
    Description     string         `gorm:"type:varchar(500);comment:描述"`
    CronExpr        string         `gorm:"type:varchar(100);not null;comment:cron 表达式"`
    TargetType      string         `gorm:"type:varchar(20);not null;comment:目标类型(single_node/tag)"`
    TargetID        string         `gorm:"type:uuid;not null;comment:目标ID(node_id 或 tag_id)"`
    CommandName     string         `gorm:"type:varchar(100);not null;comment:命令名称"`
    CommandParams   datatypes.JSON `gorm:"type:jsonb;default:'{}';comment:命令参数"`
    WaitResponse    bool           `gorm:"default:false;comment:是否同步等待响应"`
    WaitTimeoutSec  int            `gorm:"default:30;comment:同步等待超时(秒)"`
    Enabled         bool           `gorm:"default:true;comment:是否启用"`
    MaxRetries      int            `gorm:"default:3;comment:失败后最大重试次数"`
    LastRunAt       *time.Time     `gorm:"comment:上次运行时间"`
    RetryIntervalSec int           `gorm:"default:60;comment:重试间隔(秒)"`

    Records         []EdgeScheduledTaskRecord `gorm:"foreignKey:TaskID"`
}
```

### 2.3 EdgeScheduledTaskRecord 执行记录

```go
type EdgeScheduledTaskRecord struct {
    BaseModel
    TaskID          string         `gorm:"type:uuid;not null;index;comment:任务ID"`
    NodeID          string         `gorm:"type:uuid;not null;index;comment:目标节点ID"`
    Status          string         `gorm:"type:varchar(20);not null;default:pending;comment:状态(pending/dispatched/success/failed)"`
    TraceID         string         `gorm:"type:varchar(50);comment:MQTT trace_id"`
    RequestPayload  datatypes.JSON `gorm:"type:jsonb;comment:下发请求参数"`
    ResponsePayload datatypes.JSON `gorm:"type:jsonb;comment:响应内容"`
    ErrorMessage    string         `gorm:"type:text;comment:错误信息"`
    RetryCount      int            `gorm:"default:0;comment:已重试次数"`
    ExecutedAt      *time.Time     `gorm:"comment:执行时间"`
    CompletedAt     *time.Time     `gorm:"comment:完成时间"`
}
```

### 2.4 命令名称常量

```go
// go 侧常量枚举，与 C++ CommandDispatcher 对齐
const (
    ScheduledTaskCmdStartStream    = "start_stream"
    ScheduledTaskCmdStopStream     = "stop_stream"
    ScheduledTaskCmdStartPlayback  = "start_playback"
    ScheduledTaskCmdStopPlayback   = "stop_playback"
    ScheduledTaskCmdStreamStatus   = "stream_status"
    ScheduledTaskCmdSelfCheck      = "self_check"
    ScheduledTaskCmdFaceLibrary    = "face_library"
    ScheduledTaskCmdAlgoWarmup     = "algo_warmup"
    ScheduledTaskCmdFaceEmbedding  = "face_embedding"
)
```

## 3. 分层组件

### 3.1 Repository 层

| Repository | 职责 |
|---|---|
| `EdgeNodeTagRepository` | 标签 CRUD，查询标签下的节点列表 |
| `EdgeScheduledTaskRepository` | 计划任务 CRUD，按条件查询待执行任务 |
| `EdgeScheduledTaskRecordRepository` | 执行记录 CRUD，记录清理，分页查询 |

### 3.2 Service 层

**EdgeNodeTagService** — 标签管理，无特殊业务逻辑，薄透传。

**EdgeScheduledTaskService** — 核心业务逻辑：

| 方法 | 职责 |
|---|---|
| `Create/Update/Delete/GetByID/List` | CRUD 透传 |
| `ToggleEnabled` | 启用/禁用 |
| `ExecuteTask(task)` | 执行任务：解析目标节点 → 创建记录 → MQTT 下发 |
| `RetryRecord(recordID)` | 重试失败的执行记录 |
| `DispatchCommand(ctx, nodeID, cmdName, params)` | 封装 MQTT 命令下发 |
| `CleanupRecords()` | 清理过期/超量记录 |

### 3.3 Handler 层

**EdgeNodeTagHandler** — CRUD + 标签与节点的关联管理
- `GET /edge-node-tags`
- `POST /edge-node-tags`
- `PUT /edge-node-tags/:id`
- `DELETE /edge-node-tags/:id`
- `GET /edge-node-tags/:id/nodes` — 查询标签下的节点
- `PUT /edge-nodes/:id/tags` — 为节点打标签

**EdgeScheduledTaskHandler** — 计划任务管理
- `GET /edge-scheduled-tasks`
- `POST /edge-scheduled-tasks`
- `GET /edge-scheduled-tasks/:id`
- `PUT /edge-scheduled-tasks/:id`
- `DELETE /edge-scheduled-tasks/:id`
- `PUT /edge-scheduled-tasks/:id/toggle`
- `GET /edge-scheduled-tasks/:id/records` — 执行记录
- `POST /edge-scheduled-tasks/:id/records/:recordId/retry`

### 3.4 Asynq Task 层

**EdgeScheduledTaskPatrol** — 计划任务巡检（1 分钟周期）：

```text
1. 查询所有 enabled 的 EdgeScheduledTask
2. 对每个任务，判断 cron 表达式是否匹配当前时间
3. 解析目标节点列表：
   - single_node → 直接取节点（检查是否 online/enabled）
   - tag → 查询 tag 关联的所有 enabled 节点
4. 对每个目标节点：
   a. 创建执行记录 (status=pending)
   b. 构建 MQTT payload (trace_id + command params)
   c. 发布 MQTT 消息
   d. 更新记录为 dispatched
   e. 如 wait_response=true，启动 goroutine 等待响应
   f. WebSocket 广播执行事件
5. 更新 task.last_run_at
```

**EdgeScheduledTaskRecordCleanup** — 执行记录清理（每日一次）：
```text
1. 清理 created_at < 90 天前的记录
2. 对每个任务，保留最近的 100 条记录
```

## 4. MQTT 命令下发流程

### 4.1 Fire-and-forget（默认）

```go
func (s *EdgeScheduledTaskService) dispatchFireForget(ctx, nodeID, commandName string, params map[string]interface{}) error {
    traceID := uuid.New().String()
    payload := map[string]interface{}{
        "trace_id": traceID,
    }
    // 合并命令参数
    for k, v := range params {
        payload[k] = v
    }
    payloadBytes, _ := json.Marshal(payload)
    topic := fmt.Sprintf("aivision/edge/%s/cmd/%s", nodeID, commandName)
    token := s.mqttClient.Publish(topic, 1, false, payloadBytes)
    token.Wait()
    return token.Error()
}
```

### 4.2 同步等待（wait_response=true）

```go
func (s *EdgeScheduledTaskService) dispatchWaitResponse(ctx, nodeID, commandName string, params map[string]interface{}, timeout time.Duration) (string, error) {
    traceID := uuid.New().String()
    payload := buildPayloadWithTraceID(traceID, params)
    payloadBytes, _ := json.Marshal(payload)
    topic := fmt.Sprintf("aivision/edge/%s/cmd/%s", nodeID, commandName)
    token := s.mqttClient.Publish(topic, 1, false, payloadBytes)
    token.Wait()
    if token.Error() != nil {
        return "", token.Error()
    }
    return s.syncManager.Wait(ctx, traceID, timeout)
}
```

复用现有的 `mqttsync.MqttSyncManager`（Redis Pub/Sub）和 edge 节点的 response topic 订阅机制。

## 5. 路由注册

在 `app/internal/router/router.go` 现有 `registerEdgeNodeRoutes` 或新建 `registerEdgeScheduledTaskRoutes` 中加入：

```go
// 标签路由
v1.GET("/edge-node-tags", ...)
v1.POST("/edge-node-tags", ...)
v1.PUT("/edge-node-tags/:id", ...)
v1.DELETE("/edge-node-tags/:id", ...)

// 计划任务路由
v1.GET("/edge-scheduled-tasks", ...)
v1.POST("/edge-scheduled-tasks", ...)
v1.PUT("/edge-scheduled-tasks/:id", ...)
v1.DELETE("/edge-scheduled-tasks/:id", ...)
v1.PUT("/edge-scheduled-tasks/:id/toggle", ...)
v1.GET("/edge-scheduled-tasks/:id/records", ...)
v1.POST("/edge-scheduled-tasks/:id/records/:recordId/retry", ...)

// 节点标签关联
v1.PUT("/edge-nodes/:id/tags", ...)
```

## 6. Wire DI 依赖注入

### 新增 Provider

```
repositories:
  - EdgeNodeTagRepository
  - EdgeScheduledTaskRepository
  - EdgeScheduledTaskRecordRepository

services:
  - EdgeNodeTagService
  - EdgeScheduledTaskService

handlers:
  - EdgeNodeTagHandler
  - EdgeScheduledTaskHandler

asynq task:
  - EdgeScheduledTaskPatrol (handlers + periodic registration)
```

在 `app/internal/router/deps.go` 中新增 provider 函数，在 `app/internal/router/wire.go` 中注册，运行 `cd app && make wire` 生成。

## 7. WebSocket 事件

```
scheduled_task.executed   → { task_id, record_id, node_id, command_name, status, executed_at }
scheduled_task.completed  → { task_id, record_id, node_id, command_name, status, response, completed_at }
```

## 8. 迁移方案

- 新增模型通过 GORM AutoMigrate 自动创建表（现有 `server/` 已有迁移入口）
- 无破坏性变更，可安全回滚（删除表）
- 记录清理的 Asynq task 不会影响已有数据的完整性

## 9. 安全考虑

- 命令参数透传 — 不对传参内容做深度校验（由 Engine 侧 CommandDispatcher 负责）
- 目标节点校验 — 只向 online/enabled 的节点下发命令
- CRUD 操作需要管理员权限（复用现有 RBAC 中间件）
