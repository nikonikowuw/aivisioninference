# 统一节点前端契约与凭证治理

## Goal

统一边缘节点 API 与 React 数据契约，并补齐节点禁用、删除、token 轮换和实时状态展示的运维闭环。

## Requirements

- 统一节点硬件字段的 API 结构，消除后端平铺字段与前端 `hardware_info` 的不一致。
- 对齐 `hal_platform`、CPU、GPU、总内存、算法运行状态和部署响应类型。
- 节点列表和详情页正确展示真实数据，不依赖 legacy fallback 字段。
- WebSocket 心跳更新优先局部更新对应节点，避免每次心跳全量刷新列表。
- 定义节点 token 的轮换与吊销接口及审计行为。
- 心跳认证除 JWT 签名外，还必须校验节点存在、启用且 token 当前有效。
- 禁用或删除节点后，旧 token 不得使节点重新上线或获取部署任务。
- 保持创建节点时 token 只展示一次的使用体验。

## Acceptance Criteria

- [ ] 列表和详情正确显示 HAL、CPU、GPU、总内存和算法运行状态。
- [ ] TypeScript 类型与实际 Go API 响应一致，前端构建无类型规避。
- [ ] 单节点心跳不会触发无关节点列表的全量重复请求。
- [ ] token 可轮换，轮换后旧 token 立即失效。
- [ ] 禁用和删除节点后旧 token 的心跳请求被拒绝。
- [ ] 凭证操作有权限控制、审计记录和后端测试。

## Dependencies

- 后端展示字段依赖心跳状态语义稳定。
- 节点可用性与禁用行为需兼容任务路由和算法生命周期子任务。

## Implementation Status

### 前端 API 契约对齐
- Go `EdgeNode` 模型 JSON 字段 `hal_platform` 与前端的 `platform` 字段名对齐 ✅
  - 前端接口：`platform` → `hal_platform` ✅
  - 列表页展示：从 `hardware_info.platform || platform` 回退改为 `hal_platform` ✅
  - 详情页展示：从 `hardware_info.platform` 改为 `hal_platform` ✅
- Go `EdgeNode` 模型含 `cpu_model`、`gpu_model`、`total_memory`、`cpu_usage`、`memory_usage` 等平铺字段 ✅
- 前端 `HardwareInfo` 接口已定义，但 Go 响应返回的是模型本身（而非 DTO），字段直接平铺在根级 ✅

### 类型一致性
- 前端 `EdgeNode` 接口字段与 Go 模型 JSON 标签对齐（`hal_platform`、`engine_version`、`uptime` 等）✅
- `runtime_error` 字段已加入前端接口 ✅

### WebSocket 局部更新
- `HandleHeartbeat` 广播 `edge-node-status` 事件含 `node_id`，前端可通过 `node_id` 局部更新 ✅
- `BroadcastMetricsEvent` 广播 `edge-node-metrics` 事件含 `node_id`，前端局部更新 ✅
- 每次心跳不会触发全量列表刷新（前端只更新对应节点的指标和状态）✅

### Token 禁用/吊销
- 禁用节点：`HandleHeartbeat` 早期返回空响应，不进入调度池 ✅
- 删除节点：`findNodeByID` 失败，心跳被拒绝 ✅
- Token 轮换：暂未实现（需要 GenerateNodeToken + 旧 token 黑名单接口）
- Token 吊销：`ParseNodeToken` 未检查 Redis 黑名单（与用户 access token 不同）
  - 环节：`EdgeNodeMiddleware.AuthNode()` → `jwtManager.ParseNodeToken()` 仅验证签名+过期
  - 增量实现：在 ParseNodeToken 中添加黑名单检查，在节点禁用/删除时写入黑名单

### 凭证操作审计
- 节点创建/更新/删除 API 均经过 Gin middleware 认证和权限校验 ✅
- 系统操作日志（操作审计）已覆盖基础运维操作 ✅

### 未实现
- Token 轮换接口（建议：POST /edge-nodes/{id}/rotate-token → 生成新 token + 黑名单旧 token）
- Token 黑名单机制（需要 Redis + ParseNodeToken 改造）
- 禁用/删除节点时自动黑名单旧 token

## Notes

- 后端展示字段依赖心跳状态语义稳定。
- 节点可用性与禁用行为需兼容任务路由和算法生命周期子任务。
- Token 黑名单建议在独立 PR 中增量实现，涉及 `ParseNodeToken` + `EdgeNodeMiddleware.AuthNode()` + `EdgeNodeService` 的三层改动。
