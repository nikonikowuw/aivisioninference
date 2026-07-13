# 修复边缘节点运行闭环

## Goal

作为父任务协调边缘节点运行闭环的分阶段修复，使控制面记录的节点、算法和任务状态准确反映 C++ Engine 的实际状态，并确保用户选择或系统推荐的目标节点真正接收推理指令。

父任务不直接承担实现；实现由下列子任务完成，父任务负责跨子任务契约、依赖顺序和最终集成验收。

## Child Tasks

| Order | Child task | Priority | Scope |
| --- | --- | --- | --- |
| 1 | `07-12-edge-node-task-routing` | P0 | AI 任务目标节点寻址、节点能力校验、MQTT 命令契约 |
| 2 | `07-12-edge-node-media-scheduler` | P1 | 媒体容量指标、预览流准入、节点选择与粘性分配 |
| 3 | `07-12-edge-node-liveness-recovery` | P0 | 心跳超时、MQTT LWT、任务暂停与 Engine 确认恢复 |
| 4 | `07-12-edge-node-heartbeat-consistency` | P1 | 真实事务、状态字段语义、共享 Hub 与幂等更新 |
| 5 | `07-12-edge-node-algorithm-lifecycle` | P1 | 部署幂等、超时恢复、自动重试、失败原因与卸载语义 |
| 6 | `07-12-edge-node-contract-security` | P2 | 前端 API 契约、节点凭证禁用/轮换、局部实时更新 |

依赖规则：任务路由先建立显式 `node_id` 控制契约；媒体调度器和任务恢复均依赖该契约。心跳一致性应在离线恢复最终集成前完成；前端契约与凭证治理在相关后端契约稳定后完成。算法生命周期可以与任务路由并行，但最终需要共同通过父任务集成测试。

## Background

当前边缘节点功能已经具备管理端 CRUD、节点 JWT、HTTP 心跳、算法包拉取、MQTT 控制和节点推荐等基础能力，但代码分析确认存在以下运行时断点：

- AI 任务保存了 `target_node_id`，但启动流的 MQTT topic 使用设备 ID，目标节点未参与实际路由。
- `edge_node:status_check` handler 已注册但没有周期投递，`heartbeat_check_interval_sec` 未生效。
- 算法自动重试既未接入 worker mux，也未注册周期任务。
- 健康心跳会恢复节点下所有 `suspended` 任务，无法区分节点离线暂停与人工或业务暂停。
- 节点恢复时只更新任务数据库状态，没有确保 Engine Pipeline 被重新创建。
- 心跳事务没有把 GORM transaction handle 传给参与更新的 repositories，无法保证原子性。
- 算法部署在响应送达 Engine 前变为 `downloading`，响应丢失后可能永久卡住。
- 算法“卸载”只删除控制面记录，不会卸载 Engine 中的算法或删除节点文件。
- 后端平铺节点硬件字段与前端 `hardware_info` 类型不一致。
- 健康心跳会清空管理员填写的 `remark`。
- Asynq 边缘节点状态任务使用独立 `ws.Hub`，与应用 WebSocket Hub 不共享。

## Requirements

### R1. Target-node routing

- AI 推理任务启动、停止、查询和状态协调必须使用 `target_node_id` 路由到对应 Engine。
- 设备或通道 ID 继续作为流与业务资源标识，不得再承担节点寻址职责。
- 创建、更新和启动任务时必须校验目标节点在线、启用、未超载，并已安装任务所需算法。

### R2. Node liveness

- 按配置的检查间隔执行节点心跳超时扫描。
- HTTP 心跳超时和 MQTT LWT 必须收敛到同一套离线处理逻辑。
- 离线状态持久化、任务处理和 WebSocket 通知必须使用应用共享的依赖实例。
- 重复离线事件必须保持幂等。

### R3. Task suspension and recovery

- 仅自动恢复明确因节点离线而暂停的任务。
- 人工暂停、调度暂停和其他错误导致的暂停不得被普通心跳恢复。
- 节点恢复后，控制面必须重新确认或重建 Engine 运行状态，再将任务标记为 `running`。
- 恢复失败时保留可诊断的状态和错误原因。

### R4. Heartbeat consistency

- 节点字段、算法清单和任务状态变更必须在真实的数据库事务中执行，或采用明确可恢复的分步一致性机制。
- 心跳不得覆盖管理员维护的备注字段；运行时错误使用独立字段或明确的运行状态载荷。
- 禁用节点不得继续进入在线调度池或触发任务自动恢复。

### R5. Reliable algorithm deployment

- 部署命令必须具备可重试、幂等和超时恢复能力。
- `pending`、`downloading`、`installed`、`failed` 状态必须有明确的转换条件和超时策略。
- 自动重试 handler 和周期调度必须真正接入运行时，并遵循最大次数和退避策略。
- Engine 安装失败原因应完整上报并持久化，不能只保存通用错误。

### R6. Algorithm removal semantics

- 管理端和 API 不得把“只删除控制面关联记录”描述为已从节点卸载。
- 最终行为应选择并实现以下一种明确语义：Engine 热卸载并清理文件，或降级为“取消管理/重启后卸载”并在 UI 中准确表达。

### R7. API and frontend contract

- 统一 Go API 与 React 类型中的节点硬件、平台、算法状态和部署响应字段。
- 列表和详情页能够显示真实的 HAL 平台、CPU、GPU、内存和算法运行状态。
- WebSocket 更新应按节点局部更新，避免每次心跳导致无必要的全列表刷新。

### R8. Security and operability

- 明确节点 token 的禁用、轮换和吊销策略。
- 禁用或删除节点后，旧 token 不得继续执行有效心跳或重新加入调度池。
- 对心跳失败、状态转换、部署重试和任务恢复提供结构化日志。

## Acceptance Criteria

- [ ] AI 任务的所有 Engine MQTT 命令发送到 `aivision/edge/{target_node_id}/...`，消息载荷仍包含设备/通道标识。
- [ ] 创建和启动任务时，离线、禁用、满载或缺少目标算法的节点会被拒绝，并返回可定位的业务错误。
- [ ] 普通预览流根据解码、编码和带宽容量选择节点，并持久化稳定的流节点归属；不得使用单一 Pipeline 数量代替媒体负载。
- [ ] 边缘节点状态检查按照配置运行，超过超时时间后节点自动变为 `offline`。
- [ ] MQTT LWT 与心跳超时执行相同的任务暂停和通知行为。
- [ ] 只有带有“节点离线”暂停原因或等价机器可判定标记的任务会在节点恢复后自动恢复。
- [ ] 节点恢复后任务仅在 Engine 确认 Pipeline 启动成功后变为 `running`。
- [ ] 心跳处理中任一数据库步骤失败时，不会留下部分提交的节点、算法或任务状态。
- [ ] 管理员备注在健康和错误心跳后均保持不变。
- [ ] 算法部署响应丢失或节点中断后，可以从超时的 `downloading` 状态自动恢复。
- [ ] 算法自动重试在实际 Asynq worker 和 scheduler 中运行，并有覆盖退避与最大次数的测试。
- [ ] 算法失败的 Engine 原始错误信息能够在节点详情页查看。
- [ ] 算法删除操作的后端行为与管理端文案一致，不再产生“已卸载但 Engine 仍加载”的误导。
- [ ] 节点列表和详情正确展示后端上报的 `hal_platform`、CPU、GPU、总内存和算法运行状态。
- [ ] 禁用或删除节点后，原节点 token 无法使节点恢复在线或获取待部署算法。
- [ ] Go 单元测试、Engine 测试和 React 构建通过，关键跨层流程有新增回归测试。

## Constraints

- 保持 Go Handler -> Service -> Repository 分层，不绕过 Wire 手动创建业务依赖。
- 控制命令继续使用 MQTT，节点心跳继续使用 HTTP；除非后续设计明确批准协议迁移。
- 不直接修改生成的 `wire_gen.go` 或 FlatBuffers 生成代码。
- 数据库和协议变更需要兼容已有节点记录与现有 Engine 配置。

## Out Of Scope

- 重写完整的 Engine Pipeline 或算法 ABI。
- 将 MQTT/HTTP 全面替换为新的通信协议。
- 与边缘节点闭环无关的设备、GB28181 或智能记录功能重构。

## Notes

- 这是跨 Go、React、C++、MQTT 与 Asynq 的父任务。每个复杂子任务在启动实现前必须补充自己的 `design.md` 和 `implement.md`。
- 父任务仅在全部子任务完成且跨层集成验收通过后归档。
