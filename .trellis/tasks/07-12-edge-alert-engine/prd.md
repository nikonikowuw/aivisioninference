# Alert Engine & Notification

## Goal

在边缘节点监控指标采集（Phase 1）之上构建告警引擎，支持自定义告警规则配置、阈值评估、事件生命周期管理和多渠道通知推送。

## Requirements

### 告警规则管理

- 告警规则支持 CRUD（创建/读取/更新/删除）操作。
- 规则支持按节点绑定（`node_id`）或全局（`node_id IS NULL`，适用于所有节点）。
- 配置字段：名称、指标类型、比较操作符、阈值、持续时长（秒）、静默期（分钟）、启用/禁用、通知渠道列表。
- 支持的指标类型：`cpu_usage`、`memory_usage`、`disk_usage`、`node_offline`、`node_error`、`temperature`。
- 支持的比较操作符：`>`、`>=`、`<`、`<=`、`==`。

### 告警规则评估

- 每次心跳处理后自动触发规则评估（通过 `EvaluateAfterHeartbeat` 集成到 `HandleHeartbeat`）。
- 规则必须满足持续时长才触发（`duration_seconds` 时间内持续超阈值）。
- 指标回落到阈值以下后自动解析（`resolved` 状态）。
- 节点从离线/错误恢复上线时自动评估节点回线告警（`EvaluateNodeBackOnline`）。

### 告警事件生命周期

- 触发时创建 `firing` 状态事件，记录指标值、触发时间、关联规则和节点。
- 指标回落后自动转为 `resolved`，记录恢复时间。
- 运维可手动确认事件（`acknowledged`）。
- 事件状态转换：`firing` → `resolved` / `acknowledged`。

### 静默期机制

- 同规则同节点触发后 `silence_minutes` 内不重复通知。
- 静默期状态从数据库恢复，进程重启后静默期不丢失。
- 静默期过期后重新触发时恢复通知。

### 通知渠道

- 6 个内置通知提供者：Webhook（JSON POST）、Email（SMTP）、Telegram Bot、钉钉 Webhook、飞书 Webhook、企业微信 Webhook。
- 通知提供者通过 `Notifier` 接口注册，支持运行时新增。
- 规则可通过 `notify_channels` JSON 字段选择使用的渠道。

### 前端管理页面

- 告警规则列表页：展示规则名称、指标类型、阈值、状态（启用/禁用）、绑定节点。
- 告警规则表单页：创建/编辑规则，含指标类型、操作符、阈值、持续时长、静默期、通知渠道选择器。
- 告警事件历史列表：展示状态（firing/resolved/acknowledged）、指标值、触发/恢复时间、支持按节点/规则/状态筛选。

## Architecture

```
Edge Node Heartbeat
       │
       ▼
HandleHeartbeat()
       │
       ├─ Persist metrics → edge_node_metrics
       ├─ Broadcast via WebSocket
       └─ EvaluateAfterHeartbeat()
              │
              ▼
         AlertEngine
              │
              ├─ ListActiveByNode(nodeID)
              ├─ For each rule:
              │    ├─ extractMetricValueForRule()
              │    ├─ evaluateRule()
              │    │    ├─ compareValue() → threshold met?
              │    │    ├─ checkDuration() → persisted long enough?
              │    │    ├─ Create/fire event
              │    │    └─ Send notification via NotifierRegistry
              │    └─ Auto-resolve if threshold no longer met
              │
              └─ EvaluateNodeBackOnline()
                     └─ Resolve node_offline/node_error events
```

## Data Model

### alert_rules
| 字段 | 类型 | 说明 |
|------|------|------|
| id | UUID | 主键 |
| name | varchar(255) | 规则名称 |
| node_id | uuid? | 关联节点（NULL = 全局规则） |
| metric_type | varchar(50) | 指标类型 |
| operator | varchar(5) | 比较操作符 |
| threshold | double | 阈值 |
| duration_seconds | int | 持续时长（秒） |
| silence_minutes | int | 静默期（分钟） |
| enabled | bool | 启用/禁用 |
| notify_channels | jsonb | 通知渠道列表 |

### alert_events
| 字段 | 类型 | 说明 |
|------|------|------|
| id | UUID | 主键 |
| rule_id | UUID | 关联规则 |
| node_id | UUID | 关联节点 |
| metric_value | double | 触发值 |
| status | varchar(20) | firing/resolved/acknowledged |
| fired_at | timestamptz | 触发时间 |
| resolved_at | timestamptz? | 恢复时间 |
| acknowledged_by | varchar(255)? | 确认人 |
| acknowledged_at | timestamptz? | 确认时间 |
| notify_sent | bool | 是否已通知 |
| notify_sent_at | timestamptz? | 通知时间 |

## Implementation Status

后端：AlertRule 模型/仓库/服务/处理器 ✅
后端：AlertEvent 模型/仓库/服务/处理器 ✅
后端：AlertEngine 评估引擎（阈值/持续时长/静默期） ✅
后端：Notifier 接口 + 6 个实现（Webhook/Email/Telegram/钉钉/飞书/企微） ✅
后端：NotifierConfig + InitDefaultNotifiers ✅
后端：Wire DI 集成（deps.go） ✅
后端：HandleHeartbeat 集成（EvaluateAfterHeartbeat 回调） ✅
后端：静默期持久化（RestoreSilenceState 启动时恢复） ✅
前端：告警规则列表 + 表单页 ✅
前端：告警事件历史列表 ✅

## Not Implemented (Future)

- 告警升级策略（连续触发升级通知频率/渠道）。
- 告警聚合（相同规则的多节点告警合并通知）。
- 基于事件响应的自动化动作（如重启引擎、拉起进程）。
- 告警静默规则（按时间窗口屏蔽）。
