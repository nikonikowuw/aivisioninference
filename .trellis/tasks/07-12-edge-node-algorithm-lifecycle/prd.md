# 完善算法部署与卸载生命周期

## Goal

使算法包从待部署、下载、安装、失败、重试到移除的生命周期具备幂等性、超时恢复能力和准确的用户语义。

## Requirements

- 定义 `pending`、`downloading`、`installed`、`failed` 的合法状态转换和时间戳。
- 避免在 Engine 确认接收前不可逆地消费部署任务；支持响应丢失后的重新下发。
- 为长期停留在 `downloading` 的部署建立超时恢复策略。
- 将算法自动重试 handler 接入 worker mux，并由 scheduler 按明确频率投递。
- 保留退避、最大重试次数和人工重试能力。
- Engine 心跳上报真实安装失败原因，控制面完整持久化并通过 API 返回。
- 明确移除语义：优先实现 Engine 热卸载与文件清理；若技术验证不支持，则改为准确的"取消管理/重启后移除"语义并阻止误导性状态。
- 重复部署、重复回执和重复移除必须幂等。

## Acceptance Criteria

- [ ] 心跳响应丢失后部署能够自动重新下发，不会永久卡在 `downloading`。
- [ ] 自动重试在实际 worker 和 scheduler 中运行，遵守退避和最大次数。
- [ ] Engine 原始失败原因能够在 API 和管理端展示。
- [ ] 重复部署或重复回执不会产生并发下载和错误状态倒退。
- [ ] 算法移除后的 Engine 实际状态与 API/UI 文案一致。
- [ ] Go 与 Engine 测试覆盖成功、失败、超时、重试和移除路径。

## Dependencies

- 可与任务路由并行实施。
- 最终节点可用性查询需要与 `07-12-edge-node-task-routing` 的算法校验规则集成。

## Implementation Status

### 状态机与幂等性
- 状态常量定义：`pending` → `downloading` → `installed` / `failed`，含 `ErrorMessage` / `RetryCount` / `LastRetryAt` 字段 ✅
- 重复部署检测：`EdgeNodeService.DeployAlgorithm()` 检查状态——已 `installed` 返回错误，`pending`/`downloading` 返回"正在下发中" ✅
- 失败部署通过 `ResetDeployment()` 重置为 `pending` 后重新下发 ✅
- `SyncInstalled()` 在 HandleHeartbeat 中批量同步 Engine 上报的已安装列表 ✅

### 自动重试
- `EdgeNodeAlgorithmRetryTask`：查找 `failed` 且 `retry_count < 3` 的部署 ✅
- 指数退避：5m / 10m / 20m（`calculateRetryDelay`） ✅
- 跳过离线节点的重试 ✅
- 每 5 分钟周期调度（已注册到 `asynq.Scheduler`） ✅
- 人工重试：管理端 `retryAlgo` 按钮 ✅

### 错误原因持久化
- `EdgeNodeAlgorithm.ErrorMessage` 存储 Engine 上报的原始失败原因 ✅
- 通过 `ListAlgorithms` API 返回，前端节点详情页展示 ✅

### 移除语义
- 设计选择："取消管理/重启后移除"（当前 Engine 未实现热卸载） ✅
- 后端：仅删除数据库关联记录，不向 Engine 发送卸载命令 ✅
- 前端：6 语言文案统一为"移除管理"而非"卸载"，避免误导 ✅

### 测试覆盖
- `TestEdgeNodeAlgorithmRepository_Operations`：状态转换（pending→downloading→installed） ✅
- `TestEdgeNodeAlgorithmRepository_SyncInstalledBatch`：批量同步 ✅
- `TestEdgeNodeAlgorithmRetryTask_handleAlgorithmRetry`：重试逻辑 ✅
- `TestEdgeNodeService_Lifecycle`：部署→心跳确认→错误心跳→重试流程 ✅

### 未实现
- Engine 热卸载（需要 Engine 侧 MQTT 卸载命令和 SO 文件清理）
- `downloading` 超时自动恢复（当前依赖心跳失败后自动重试的间接路径）
