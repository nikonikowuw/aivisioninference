# 打通 AI 任务目标节点路由

## Goal

让 AI 任务选择的 `target_node_id` 成为所有推理流控制命令的真实节点寻址依据，同时保持设备/通道 ID 作为流资源标识。

## Background

- `AIVisionTask` 已持久化 `target_node_id`，但当前 `StreamManager` 只把它放入消费者 metadata，没有传入 `EngineClient`。
- `MqttEngineClient` 当前使用设备 ID 构造 `aivision/edge/{id}/cmd/...` topic，而 Engine 按节点 ID 订阅。
- 普通视频预览与 AI 推理共用 `StreamManager` 和 `EngineClient`。
- `Device` 与 `MediaStream` 当前都没有节点归属字段，因此非 AI 预览流无法从现有持久化数据推导目标节点。
- 当前 `current_load` 只是 Engine Pipeline 数量，不区分预览、推理或混合流；CPU、内存使用率仍为占位值，现有指标不足以判断预览节点负载。
- 普通预览的容量采集和自动选址由兄弟子任务 `07-12-edge-node-media-scheduler` 负责。

## Requirements

- 扩展 EngineClient 流控制契约，明确区分目标节点 ID 与设备/通道 ID。
- 所有流控制 API 接受显式 `node_id`；AI 任务启动、停止和状态查询使用任务的 `target_node_id`。
- MQTT 消息载荷继续携带设备/通道 ID、任务 ID 和算法参数。
- AI 任务创建、更新和启动时校验节点存在、在线、启用、未达到 `max_load`，且已安装所选算法。
- 节点推荐结果和手工选择节点使用同一套后端可用性规则。
- 修复受影响的 StreamManager 状态键和响应关联逻辑，避免不同节点上的相同设备标识互相污染。

## Technical Notes

- 显式节点寻址首先落在 `EngineClient` 与 MQTT topic 边界；设备/通道 ID 继续用于 Engine Pipeline、ZLM stream 和业务载荷。
- `StreamManager` 增加显式节点路由入口并使用 `(node_id, device_id)` 复合键。现有无节点调用保留短期兼容入口，后续由媒体调度器接入持久化节点归属后迁移，兼容入口不得继续用于 AI 任务。
- 节点可用性由共享服务统一判定，推荐节点和手工节点校验必须复用同一候选规则：在线、启用、`current_load < max_load`、算法部署状态为 `installed`。
- MQTT JSON 保持滚动升级兼容，不修改 FlatBuffers schema；新增或修正的 `task_id`、`device_id` 和 `trace_id` 均保留在命令载荷中。

## Out Of Scope

- 普通预览流的容量准入、`MediaStream.node_id` 持久化、节点粘性和离线迁移；这些由 `07-12-edge-node-media-scheduler` 的后续步骤负责。
- 节点离线后的 AI 任务暂停与恢复状态机；由 `07-12-edge-node-liveness-recovery` 负责。
- 重写 Engine Pipeline 标识、ZLM stream 命名或算法 ABI。

## Acceptance Criteria

- [ ] 任务发往 `target_node_id` 对应的 MQTT topic，而不是设备 ID topic。
- [ ] Engine 能从命令载荷中获得正确的设备/通道 ID 并创建对应 Pipeline。
- [ ] 离线、禁用、满载或未安装算法的节点不能创建或启动推理任务。
- [ ] 推荐节点和任务启动校验不会产生规则不一致。
- [ ] Start/Stop/Status 及多节点场景有 Go 单元测试和 MQTT topic 回归测试。

## Dependencies

- 无前置子任务，是父任务的首个 P0 实现项。
- `07-12-edge-node-liveness-recovery` 的恢复命令依赖本任务提供的节点寻址契约。
- `07-12-edge-node-media-scheduler` 依赖本任务提供的显式节点寻址契约，并负责普通预览节点选择。

## Notes

- 实现前需要 `design.md` 和 `implement.md`，重点记录 EngineClient 接口兼容方案。
