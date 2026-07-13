# AI 任务目标节点路由设计

## 设计目标

把节点寻址与流资源标识彻底分离：`node_id` 只决定 MQTT 命令发送到哪个 Engine，`device_id` / channel ID 继续标识 Pipeline、ZLM stream 和设备资源，`task_id` 标识 AI 任务。AI 任务的 Start、Stop、Status 和重试入口始终使用持久化的 `target_node_id`。

## 已确认现状

- `AIVisionTask.TargetNodeID` 已持久化，但 `StartTask` 仅把它放入 metadata。
- `StreamManager` 以 `deviceID` 为唯一状态键，调用 `EngineClient` 时不传节点 ID。
- `MqttEngineClient` 用设备 ID 构造 `aivision/edge/{id}/cmd/...`；Engine 实际订阅 `aivision/edge/{node_id}/cmd/#`。
- Start 命令把 `TaskID` 错设为设备 ID；Stop 和 Status 只携带设备 ID。响应等待已使用全局唯一 `trace_id`，无需改变 Redis/MQTT 同步机制。
- 节点推荐只过滤在线、启用和算法已安装，未排除 `current_load >= max_load`；任务创建、更新和启动没有复用推荐规则。
- 普通预览、GB28181、设备探测和旧版 `InferTaskService` 尚无稳定节点来源。媒体调度器任务已定义 `MediaStream.node_id` 与容量调度接入，属于后续消费者。

## 契约设计

### EngineClient

`StreamStartRequest` 增加必填 `NodeID` 和可选 `TaskID`。所有流控制方法显式接收节点：

```go
StartStream(ctx context.Context, req StreamStartRequest) (StreamInfo, error)
StopStream(ctx context.Context, nodeID, deviceID string) error
StartPlayback(ctx context.Context, req StreamStartRequest) (string, error)
StopPlayback(ctx context.Context, nodeID, deviceID string) error
GetStreamStatus(ctx context.Context, nodeID, deviceID string) (StreamStatus, error)
```

Start 方法从 request 读取 `NodeID`；Stop/Status 用两个独立参数，避免把两个 UUID 位置写反。生产实现和所有测试 stub 同步更新。

### MQTT JSON

- topic 固定为 `aivision/edge/{node_id}/cmd/{command}`。
- payload 保留 `device_id` 和 `trace_id`。
- Start payload 的 `task_id` 使用真实 AI 任务 ID；非任务型流可回退为 device ID，保证旧 Engine 对非空 task ID 的预期。
- 不修改 FlatBuffers schema。Engine 命令处理仍以 `device_id` 创建、查询和销毁 Pipeline，因此 C++ 仅需协议回归验证，不需要改变 Pipeline key。

### StreamManager

引入不可混淆的路由值对象，例如 `StreamRoute{NodeID, DeviceID}`，并为 AI/媒体调用提供显式节点的 Acquire、Release、KeepAlive、Get 和 Status 路径。运行时状态使用稳定复合键编码 `(node_id, device_id)`，`StreamState` 保存 NodeID；所有 Stop、Status、重试和恢复从 state 读取原节点，不重新选择。

消费者键不能只使用固定字符串 `infer`。AI 消费者使用稳定的任务级 key（例如 `infer:{task_id}`），避免同节点同设备上的重复任务互相覆盖；事件增加 NodeID，使任务事件匹配同时检查目标节点和设备。

为未完成迁移的普通预览/GB28181/旧版 InferTask 调用保留短期兼容入口，但兼容入口不得被 `AIVisionTaskService` 使用，也不得伪造 `node_id=device_id`。媒体调度器后续以 `MediaStream.node_id` 接入后删除或收紧兼容入口。

## 节点可用性

抽取共享的推理节点策略组件，由节点推荐和 AI 任务校验共同使用。候选节点必须同时满足：

1. 节点存在、`status=online`、`enabled=true`。
2. `max_load > 0` 且 `current_load < max_load`。
3. 目标算法在该节点的部署记录为 `installed`。

手工节点校验返回具体业务错误；推荐接口从同一候选查询/判定中排序，按负载率升序、节点 ID 稳定打破平局。创建和更新校验配置合法性，StartTask 再次校验运行时可用性，防止状态漂移。

共享策略优先作为独立 service/helper 注入 `AIVisionTaskService` 和 `EdgeNodeService`，避免两个服务互相依赖。Repository 只提供查询，不承载业务错误映射。

## 数据流

1. 创建/更新任务时，用 `target_node_id + algo_package_id` 调用共享可用性校验。
2. StartTask 再校验节点和设备，构造包含 NodeID、DeviceID、TaskID、算法信息的启动请求。
3. StreamManager 用复合键建立或复用节点内 Pipeline 状态。
4. MqttEngineClient 将命令发布到目标节点 topic，Engine 继续用 payload 的 device ID 操作 Pipeline。
5. Stop、Status、重试和事件处理从原 StreamState/任务记录取 node ID，禁止按当前推荐结果重新路由。

## 兼容与回滚

- MQTT payload 只修正字段值并保留现有字段，支持 Go 与 Engine 滚动升级；旧 Go 仍会错误寻址，因此部署顺序应先升级控制面或同时升级。
- 本任务不迁移数据库结构，不修改生成的 FlatBuffers 文件。
- EngineClient 是共享接口，编译错误会暴露所有未更新 stub/caller；实施时先修改接口与 mock，再逐层修复。
- 回滚时可恢复旧接口实现，但不得删除任务已有的 `target_node_id` 数据。媒体调度器新增的数据列保持不动。

## 风险

- 最大风险是节点 ID 与设备 ID 参数错位；使用 request/value object、命名参数字段和 topic 单测降低风险。
- StreamManager 复合键若未覆盖事件、超时释放、重试和恢复，会留下跨节点串流；所有遍历和回调必须携带 NodeID。
- 创建校验与启动校验若实现两份规则会再次漂移；共享策略及表驱动测试是强制门禁。
