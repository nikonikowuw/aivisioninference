# 修复节点离线检测与任务恢复

## Goal

建立统一、可配置、幂等的节点离线与恢复流程，使任务状态与 Engine Pipeline 实际状态一致。

## Requirements

- 按 `heartbeat_check_interval_sec` 投递节点状态检查任务，并按 `heartbeat_timeout_sec` 判定超时。
- HTTP 心跳超时和 MQTT LWT 调用同一个离线领域服务。
- 离线处理统一完成状态持久化、运行任务暂停和 WebSocket 通知。
- 记录机器可判定的暂停来源，区分节点离线、人工暂停、调度暂停和其他错误。
- 只有因节点离线暂停的任务才允许自动恢复。
- 节点恢复时通过目标节点路由重新启动或协调 Pipeline；Engine 确认成功后才能更新为 `running`。
- 恢复失败时保持 `suspended` 或明确错误状态，并记录原因。

## Acceptance Criteria

- [ ] 周期调度器真实投递并执行边缘节点状态检查。
- [ ] MQTT LWT 和心跳超时产生一致且幂等的离线结果。
- [ ] 离线节点上的运行任务被标记为节点离线暂停。
- [ ] 人工或其他原因暂停的任务不会被心跳自动恢复。
- [ ] Engine 未确认 Pipeline 恢复时，任务不会显示为 `running`。
- [ ] 离线、重复离线、恢复成功和恢复失败均有回归测试。

## Dependencies

- 依赖 `07-12-edge-node-task-routing` 提供正确的恢复命令节点寻址。
- 与 `07-12-edge-node-heartbeat-consistency` 共同完成最终一致性验收。

## Notes

- 实现前需要 `design.md` 和 `implement.md`，并明确任务暂停来源的数据迁移方案。
