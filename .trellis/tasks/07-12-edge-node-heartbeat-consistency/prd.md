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

## Notes

- 实现前需要 `design.md` 和 `implement.md`，重点说明 Repository transaction 传播方式。
