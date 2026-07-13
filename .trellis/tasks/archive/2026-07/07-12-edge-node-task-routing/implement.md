# AI 任务目标节点路由实施计划

## 前置评审门禁

- [ ] 确认显式节点契约采用 `StreamStartRequest.NodeID/TaskID`，Stop/Playback/Status 使用独立 `nodeID, deviceID` 参数。
- [ ] 确认普通预览的节点选择、`MediaStream.node_id` 接入和离线迁移仍由 `07-12-edge-node-media-scheduler` 负责。
- [ ] 确认任务启动前保持 `planning`，由用户评审本设计后再运行 `task.py start`。

## 实施顺序

1. [ ] 扩展 `StreamStartRequest`、`EngineClient`、`MockEngineClient` 和测试 stub，明确 NodeID、DeviceID、TaskID 三种标识。
2. [ ] 修改 `MqttEngineClient` 的 Start/Stop/Playback/Status topic 与载荷构造；抽取小型 topic helper 时遵循现有复用规范，并增加节点 topic/载荷回归测试。
3. [ ] 将 `StreamManager` 状态改为 `(node_id, device_id)` 复合键，`StreamState` 保存 NodeID；修复 Acquire、Release、KeepAlive、Get、Status、超时、同步、重试和恢复路径。
4. [ ] 将消费者标识改为任务级稳定 key，并让 `DeviceEvent` 携带 NodeID，补同设备跨节点隔离和同节点多消费者测试。
5. [ ] 实现共享推理节点可用性策略：存在、在线、启用、未满载、算法已安装；Repository 查询保持数据访问职责。
6. [ ] 让 `EdgeNodeService.RecommendNode` 和 `AIVisionTaskService` 的 Create、Update、StartTask 复用同一规则，补错误映射和负载率稳定排序测试。
7. [ ] 修改 AIVisionTask Start/Stop/Restart/Status 相关调用，始终传 `target_node_id` 和真实 task ID；确认错误路径不会错误释放其他节点的同设备流。
8. [ ] 更新所有受 EngineClient 接口影响的 Go 调用方与测试 stub。非 AI 调用只做兼容迁移，不在本任务内实现媒体容量选择；把待接入点与媒体调度器步骤 8-10 对齐。
9. [ ] 检查 Engine `mqtt_control_plane`、`command_dispatcher` 和 Pipeline handlers，验证 topic 使用 node ID、payload 使用 device ID；仅在发现实际契约不兼容时修改 C++。
10. [ ] 运行聚焦测试、Go 全量单测和构建，记录媒体调度器后续接入所需的显式路由 API。

## 测试矩阵

- MqttEngineClient：Start/Stop/StartPlayback/StopPlayback/Status 均发布到目标 node topic；payload 的 device_id、task_id、trace_id 正确。
- StreamManager：相同 device ID 在两个 node 上状态、消费者、Stop 和 Status 完全隔离；重试仍回原节点。
- AIVisionTaskService：创建、更新、启动分别拒绝不存在、离线、禁用、满载和未安装算法的节点。
- 推荐一致性：推荐结果中的每个节点都能通过手工校验；不可用节点不会被推荐；负载率相同时排序稳定。
- 生命周期：Start、Stop、Restart 使用同一 target node；启动失败只更新本任务错误状态。
- Engine 协议：节点订阅 topic 与 Go 发布 topic 匹配，Engine 仍用 device ID 创建和销毁 Pipeline。

## 验证命令

```bash
cd app && go test ./internal/service ./internal/repository ./internal/handler ./internal/task
cd app && make unit-test
cd app && make build
cd engine && make test
```

如果本任务未修改 C++，`engine && make test` 仍作为 MQTT/命令契约的跨层回归检查；环境不具备依赖时必须记录未执行原因。

## 风险与回滚点

- EngineClient 接口修改是共享编译边界：先完成接口、mock 和聚焦测试，再修改 StreamManager，避免半成品扩散。
- StreamManager 复合键是最高风险点：在接入 AIVisionTask 前先通过多节点隔离测试。
- 节点可用性错误码可能需要新增 i18n key；新增时同步后端翻译资源，不返回硬编码中文。
- 不修改 `wire_gen.go`；若新增 provider，修改 `deps.go` / `wire.go` 后运行 `make wire`。
- 回滚运行时代码时保留任务的 `target_node_id` 和媒体调度器已新增的数据结构，避免数据丢失。

## 完成门禁

- [ ] 使用 `trellis-check` 完成规格、测试、构建和跨层数据流检查。
- [ ] 必要时使用 `trellis-update-spec` 固化 EngineClient 显式节点契约。
- [ ] 提交前确认未混入 `algorithms` 子模块的现有工作区改动。
- [ ] 本任务完成后通知 `07-12-edge-node-media-scheduler` 可继续步骤 8-10。
