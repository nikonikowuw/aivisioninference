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
- 明确移除语义：优先实现 Engine 热卸载与文件清理；若技术验证不支持，则改为准确的“取消管理/重启后移除”语义并阻止误导性状态。
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

## Notes

- 实现前需要 `design.md` 和 `implement.md`，热卸载能力必须先通过 Engine 源码与测试验证。
