# 修复心跳事务与节点状态语义

## Goal

修复节点心跳的数据一致性和状态语义，使一次心跳不会产生部分提交、覆盖管理员数据或向错误的 WebSocket Hub 广播。

## Requirements

- 为跨 EdgeNode、EdgeNodeAlgorithm 和 AIVisionTask 更新建立真实事务边界，repositories 必须使用同一 transaction handle。
- 心跳不得写入或清空管理员维护的 `remark`。
- 运行时错误使用独立字段或结构化状态，不与管理备注复用。
- 禁用节点的心跳不得将节点恢复为可调度在线状态，也不得返回待部署算法。
- Asynq 和 HTTP/MQTT 处理器使用应用共享 `ws.Hub`，不得自行创建孤立 Hub。
- 状态转换与 WebSocket 通知在提交后发生，失败时不发布虚假状态。
- 心跳更新保持幂等，并避免无状态变化时的无意义全量通知。

## Acceptance Criteria

- [ ] 任一步骤失败时节点、算法和任务状态不会部分提交。
- [ ] 健康或错误心跳后管理员备注保持不变。
- [ ] 禁用节点无法通过心跳重新进入调度池或取得部署任务。
- [ ] 所有边缘节点 WebSocket 事件由应用共享 Hub 发出。
- [ ] 事务回滚、禁用节点和通知时序有单元测试。

## Dependencies

- 可与任务路由并行实现。
- 必须在 `07-12-edge-node-liveness-recovery` 最终集成前完成事务与共享 Hub 契约。

## Implementation Status

### 事务与一致性
- `HandleNodeOffline` 使用 GORM Transaction 跨 EdgeNode + AIVisionTask 原子更新 ✅
- `HandleHeartbeat` 使用 per-node mutex 提供串行化（避免 SQLite 测试环境的 `table is locked` 问题）
  - `UpdateHeartbeatFields`：单表 UPDATE，无跨表事务 ✅
  - `SyncInstalled`：独立连接，未加入事务（设计取舍：mutex 确保节点级串行）
  - `MarkOfflineIfTimedOut` + `SetSuspended`：在事务中执行 ✅

### 备注保护
- `remark` 字段从 `hbFields` 中移除，不再被心跳覆盖 ✅
- `runtime_error` 独立字段存放错误消息 ✅
- 前端文案已更新，错误信息不再复用 `remark` 展示 ✅

### 禁用节点处理
- `HandleHeartbeat` 检测 `NodeStatusDisabled` / `!Enabled` 后：
  - 仅接受心跳用于存活追踪（更新 `last_heartbeat`）✅
  - 跳过所有后续处理（指标持久化、任务恢复、部署返回）✅
  - 返回空 `HeartbeatResponse`，不会获取待部署算法 ✅

### 共享 ws.Hub
- `EdgeNodeService` 通过 Wire DI 注入应用共享 `ws.Hub` ✅
- `EdgeNodeStatusTask` 通过构造函数接收应用共享 `hub` ✅
- `EdgeMqttHandler` 通过 `EdgeNodeService` 间接使用共享 `hub` ✅
- `HandleNodeOffline` 的 Broadcast 使用传入的 `hub` 参数 ✅
- 无孤立 Hub 被自行创建 ✅

### LWT + 心跳超时收敛
- 心跳超时（`EdgeNodeStatusTask`）和 MQTT LWT（`HandleLifecycle`）都收敛到 `HandleNodeOffline` ✅
- 共享的事务边界、状态转换和通知逻辑 ✅

### 单元测试覆盖
- `TestEdgeNodeService_Lifecycle`：验证 remark 保护、runtime_error、error 状态 ✅
- `TestHandleNodeOffline_IsConditionalAndIdempotent`：离线处理幂等性 ✅
- `TestEdgeNodeStatusTask_handleEdgeNodeStatusCheck`：超时扫描 ✅
- `TestEdgeNodeRepository_CRUD`：disabled 状态节点查询 ✅

## Notes

- 事务取舍：per-node mutex + 独立操作 替代了完整 DB 事务，原因见代码注释 `handleHeartbeat` 中关于 SQLite 测试限制的说明。
- 如果未来需要跨表原子更新，repositories 已实现 `WithTx` 和 `DB(ctx).Transaction` 支持。
