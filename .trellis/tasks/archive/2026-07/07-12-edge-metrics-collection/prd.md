# System Metrics Collection & Dashboard

## Goal

在现有心跳机制基础上扩展完整的系统指标采集能力，新增时序指标存储与查询 API，并实现监控仪表盘页面。这是边缘节点监控体系的基础层。

## Requirements

### C++ Engine — 指标采集扩展

- 设备监控扩展：磁盘使用率（按挂载点）、网络 I/O 总量与速率、进程/线程数、系统负载（load average 1/5/15m）、核心温度
- MetricsFlattener 模块：将 DeviceMonitor 的结构化快照平坦化为心跳 JSON 字段
- 扩展 HeartbeatReporter 的 BuildHeartbeatPayload()，包含上述新增字段
- 兼容性：已有字段位置不变，新增字段追加在末尾

### Go 控制面 — 时序存储与查询

- 新建 `edge_node_metrics` 模型和表
- 扩展 `HandleHeartbeat` 方法：心跳到达时写入 metrics 快照
- 新增指标查询 API：
  - 按节点 + 指标类型 + 时间范围分页查询
  - 支持聚合（avg/max/min）和时间窗口
- 新增节点总览 API：在线/离线/错误节点数统计
- 数据保留策略：定时清理过期记录

### React 前端 — 监控仪表盘

- 节点总览页面：6 个 stat cards（总节点/在线/离线/错误/告警数/总任务数）
- 节点列表页面增加实时指标列：CPU%、MEM%、运行时长、磁盘使用
- 节点详情页增加标签页或区域展示历史趋势图（CPU/MEM/Disk/NET 折线图）
- 时间范围选择器：1 小时 / 6 小时 / 24 小时 / 7 天

## Constraints

- C++ 采集不可引入 measurable 性能开销——读 /proc 文件系统是 µs 级操作
- 新增字段不改变已有 DTO 结构，仅追加可选字段
- 指标表使用 BRIN index 而非 B-tree——时序数据天然有序，BRIN 更高效
- API 返回数据点上限 1000 点/次，前端按时间窗口聚合降采样

## Acceptance Criteria

- [ ] C++ DeviceMonitor 采集磁盘使用率（每个挂载点）、网络 RX/TX 总量、load average、进程数、线程数、温度
- [ ] WebSocket 实时推送 edge-node-metrics 事件
- [ ] Go edge_node_metrics 表写入与查询正常工作
- [ ] 指标查询 API 支持按时间范围、聚合方式、时间窗口参数
- [ ] 数据保留 cron 按时清理过期数据
- [ ] 前端总览页正确显示节点统计数字
- [ ] 前端节点列表页显示实时 CPU%、MEM%、Uptime
- [ ] 前端详情页展示 4 个折线图 + 时间范围切换
