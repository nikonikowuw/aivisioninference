# Edge Node Monitoring — Implementation Plan

## Phase 1 — 系统指标采集与面板

### C++ Engine 扩展 (3 文件)

| 文件 | 动作 | 内容 |
|------|------|------|
| `engine/src/monitor/device_monitor.cpp` | 修改 | 扩展 disk/net/load/temp/proc 探测方法 |
| `engine/src/monitor/metrics_flattener.cpp` | **新建** | 将 DeviceSnapshot 平坦化为心跳 JSON 字段 |
| `engine/src/monitor/heartbeat_reporter.cpp` | 修改 | BuildHeartbeatPayload() 追加新字段 |

### Go 控制面新增 (10+ 文件)

| 文件 | 动作 |
|------|------|
| `app/internal/model/edge_node_metrics.go` | **新建** — 时序模型 |
| `app/internal/repository/edge_node_metrics.go` | **新建** — 带聚合的查询 |
| `app/internal/handler/edge_node_metrics.go` | **新建** — 查询+总览 API |
| `app/internal/service/edge_node_metrics_service.go` | **新建** — 查询业务逻辑 |
| `app/internal/dto/edge_node_metrics.go` | **新建** — 请求/响应 DTO |
| `app/internal/task/metrics_retention.go` | **新建** — 数据清理 Cron |
| `app/internal/service/edge_node.go` | 修改 — HandleHeartbeat 追加写 metrics |
| `app/internal/dto/edge_node.go` | 修改 — HeartbeatRequest 扩充字段 |
| `app/internal/router/edge_node.go` | 修改 — 注册指标路由 |
| `app/internal/server/deps.go` | 修改 — Wire 注册新依赖 |
| `app/internal/router/deps.go` | 修改 — Wire 注册新 handler |

### React 前端 (4 文件)

| 文件 | 动作 |
|------|------|
| `web/src/views/admin/devices/edge-nodes/overview.tsx` | **新建** — 总览页 |
| `web/src/views/admin/devices/edge-nodes/index.tsx` | 修改 — 实时指标列 |
| `web/src/views/admin/devices/edge-nodes/detail.tsx` | 修改 — 折线图区域 |
| `web/src/components/charts/MetricsTimeSeries.tsx` | **新建** — 可重用图表 |

---

**累计**: ~15 个文件，估算 ~1500-2500 行实际代码改动。

要我确认后开始实现吗？还是会直接开始？