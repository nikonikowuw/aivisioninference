# Edge Node Monitoring (Nezha-inspired)

## Goal

在 AIVisionInference 平台中实现类似哪吒监控 (Nezha Monitoring) 的边缘节点运维监控能力，覆盖系统指标采集、历史存储与可视化展示、告警引擎与多渠道通知、远程运维操作（定时任务、Web 终端）三个递进阶段。

## Requirements

### Phase 1 — 系统指标采集、存储与可视化

- C++ 推理引擎采集 Edge Node 完整系统指标：CPU 使用率、内存使用率、磁盘使用率（按挂载点）、网络 I/O（收发字节/速率）、进程/线程数、核心温度
- 通过现有 HTTP 心跳通道，将上述指标平坦化后随 `HeartbeatRequest` 上报
- Go 控制面新增 `edge_node_metrics` 时序表，按时间戳存储每次心跳的快照数据
- Go 控制面提供指标查询 API：支持按节点 + 时间段 + 指标类型分页查询
- Go 控制面实现数据保留策略（如 7 天自动清理）
- Go 控制面新增节点在线状态统计：在线率、离线时段、告警次数
- React 管理端新增监控面板页面：
  - 节点总览卡片：在线/离线/错误节点数、总任务数、总算法部署数
  - 节点详情页增加历史趋势图表：CPU 使用率、内存使用率、磁盘使用率、网络 I/O（折线图，可切换时间范围）
  - 节点列表页增加实时指标列：CPU/MEM 占用率、磁盘使用率、运行时长
  - 节点在线率统计与状态分布图

### Phase 2 — 告警引擎与通知推送

- Go 控制面新增告警规则模型 `alert_rule`，支持按节点/节点组配置阈值规则
- 支持指标类型：CPU 使用率、内存使用率、磁盘使用率、节点离线、节点错误
- 告警规则配置：
  - 指标类型 + 比较操作符（> / >= / < / <= / ==）
  - 阈值
  - 持续时长（eg. CPU > 90% 持续 5 分钟才触发）
  - 静默期（同规则触发后 N 分钟内不重复通知）
  - 启用/禁用
- 告警事件模型 `alert_event`，记录每次触发的告警包括开始时间、恢复时间、状态
- 通知渠道：
  - Webhook（通用 JSON POST）
  - 邮件（SMTP 配置）
  - Telegram Bot
  - 钉钉/飞书/企业微信 Webhook
- 告警历史列表与详情页，支持手动确认/关闭
- 告警规则 CRUD 管理页面

### Phase 3 — 远程运维

- 定时任务：
  - Go 控制面可通过 MQTT 下发 Shell 命令到 Edge Node 执行
  - C++ 引擎新增命令执行模块，执行结果通过 HTTP 回调上报
  - 定时任务支持 Cron 表达式、单次执行、重复执行
  - 任务执行历史记录与日志查看
- Web 终端：
  - Go 控制面通过 WebSocket 代理到 Edge Node 的 Shell
  - C++ 引擎新增 PTY 模块，支持远程 Shell 会话
  - 前端 Web 终端页面（基于 xterm.js 或类似组件）
  - 会话超时与安全控制

## Architecture

```
┌───────────────────────┐         ┌──────────────────────────────┐
│   C++ Inference Engine │  HTTP  │      Go Control Plane        │
│                        │◄──────►│                              │
│  DeviceMonitor         │  heartbeat │  edge_node_metrics (table) │
│   ├─ CPU/Mem/Disk      │  +     │  Metrics API                │
│   ├─ Net I/O           │  MQTT  │  Alert Engine               │
│   ├─ Temperature       │  cmds  │   ├─ Rule Evaluator         │
│   └─ Process/Thread    │        │   └─ Notifier               │
│                        │        │  Scheduled Task Mgr         │
│  Cmd Executor          │        │  Web Terminal Proxy         │
│  PTY Module            │        └──────────┬───────────────────┘
│                        │                   │ HTTP/WS
└───────────────────────┘         ┌──────────▼───────────────────┐
                                  │     React Admin Panel        │
                                  │  ├─ Monitor Dashboard        │
                                  │  ├─ Metrics Charts           │
                                  │  ├─ Alert Rules & History    │
                                  │  ├─ Scheduled Tasks          │
                                  │  └─ Web Terminal             │
                                  └──────────────────────────────┘
```

## Existing Infrastructure (to extend)

- C++ DeviceMonitor with light/expensive probe intervals, snapshot caching, multi-platform probes (Linux/RK/NVIDIA/Ascend/macOS)
- C++ HeartbeatReporter: 5s interval, JSON payload, CURL-based HTTP POST, already sends `cpu_usage`, `memory_usage`
- Go `HeartbeatRequest` DTO with `cpu_usage`, `memory_usage`, `uptime`, `current_load`, `hardware_info`
- Go `EdgeNode` model with CPUModel, GPUModel, TotalMemory, CurrentLoad, MaxLoad, Uptime, Status, LastHeartbeat
- Go WS Hub for real-time frontend updates
- React edge node list/detail/create/edit pages
- Frontend i18n across 6 languages

## Constraints

- C++ engine runs on resource-constrained devices (RK3568/RK3576): monitoring must not cause measurable CPU overhead
- All monitoring data must flow through existing HTTP heartbeat to avoid adding new connections from edge nodes
- Phase 2's alert engine must decouple notification channels from rule evaluation (provider pattern)
- Web Terminal sessions must timeout after configurable idle period (default 5 min)
- Historical metrics retention: minimum 7 days; configurable via env/config
- All timeseries queries must be paginated and bounded to prevent OOM

## Acceptance Criteria

### Phase 1
- [ ] Each heartbeat includes: CPU%, MEM%, DISK%(by mount), NET RX/TX total and rate, process/thread count, temperature
- [ ] Go control plane stores each heartbeat's metrics in `edge_node_metrics` table
- [ ] Metrics query API returns paginated, time-bounded results for any node-id + metric-type
- [ ] Retention cron removes records older than configured TTL
- [ ] Frontend dashboard shows node overview stats (total online/offline/error counts)
- [ ] Frontend detail page shows CPU/MEM/Disk/Net line charts with time range selector (1h/6h/24h/7d)
- [ ] Frontend list page shows real-time CPU%, MEM%, uptime per node
- [ ] Node online rate and status distribution chart shown on dashboard

### Phase 2
- [ ] Alert rules can be created/updated/deleted/enabled/disabled per node
- [ ] Rule evaluator runs on every heartbeat and fires events when threshold + duration met
- [ ] Fired alerts appear in alert history with status (firing/resolved/acknowledged)
- [ ] Alert auto-resolves when metric returns below threshold
- [ ] At least 2 notification channels work: Webhook + email (rest configurable by config)
- [ ] Frontend alert rule management page with CRUD + status toggle
- [ ] Frontend alert history list with filtering by node/rule/status/time range

### Phase 3
- [ ] C++ engine has command executor module that runs shell commands and returns stdout/stderr/exit code
- [ ] Go control plane can schedule one-shot and cron-based tasks on edge nodes
- [ ] Task execution results (stdout/stderr/exit_code) are stored and viewable
- [ ] Web Terminal connects from browser → Go WS → Engine PTY with real-time I/O
- [ ] Web Terminal session times out after idle period and cleans up PTY
- [ ] Frontend task management page shows task list, execution history, log viewer
- [ ] Frontend Web Terminal component with xterm.js integration

## Non-goals

- Not replacing existing inference task orchestration (stream manager, pipeline management)
- Not adding agent server/discovery (nodes are already registered manually)
- Not implementing service-level monitoring (HTTP/TCP ping between nodes) — future consideration
