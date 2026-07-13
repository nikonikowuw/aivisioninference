# 实现预览节点容量调度器

## Goal

为普通视频预览建立独立于 AI 推理负载的节点容量调度器，按每台边缘设备出厂确定的最大并发预览数执行准入、节点选择和稳定路由。

## Background

- 当前 `current_load` 是 Engine Pipeline 数量，不能准确表达节点还能启动多少路预览。
- Rockchip 的预览链路由 MPP 解码/编码承载，其他平台由各自 HAL 的硬件媒体实现承载；Go 控制面不应理解或推导底层硬件资源明细。
- 每种设备型号在出厂压力测试时即可确定最大并发预览数，运行时应把这个值作为权威容量。
- `MediaStream` 已规划持久化 `node_id`，使 Stop、Status、重试和服务重启恢复能够定位原执行节点。

## Requirements

### Engine configuration and reporting

- Engine 通过环境变量配置最大并发预览数，例如 `NIKO_ENGINE_MAX_PREVIEW_STREAMS`。
- 环境变量必须是正整数；缺失、为零或非法时，Engine 记录明确错误并上报容量无效。
- Engine 通过心跳或指标上报 `preview_capacity`、`preview_in_use` 和采集时间。
- `preview_in_use` 是当前已启用预览输出的唯一 Pipeline 数。纯推理 Pipeline 不占名额；已有推理 Pipeline 追加预览后占一个名额；重复启用同一预览不重复计数。
- 平台差异由 Engine/HAL 封装：Rockchip 使用 MPP，其他平台使用各自硬件实现，Go 只消费统一预览容量指标。

### Admission and selection

- 预览调度只使用 `preview_in_use / preview_capacity`，不复用 AI 的 `current_load / max_load`。
- 节点必须在线、启用、指标新鲜且 `preview_in_use + pending_reservations < preview_capacity`。
- 选择调度后预览占用率最低的节点；占用率相同时按节点 ID 稳定排序。
- 同一流已有活动预览时优先复用原节点，不重复占用名额。
- 容量不足时拒绝新预览，返回明确错误，不允许静默超卖。

### Assignment lifecycle

- `MediaStream` 持久化执行 `node_id`，首次成功调度后保持节点粘性。
- Start、Stop、Playback、Status、重试和服务重启恢复均使用已分配的 `node_id`。
- 并发调度必须通过数据库预留/租约或等价原子机制防止多个请求抢占最后一个预览名额。
- 启动成功后确认预留；启动失败、停止预览或租约超时后释放预留。
- 节点离线后的跨节点恢复由 liveness/recovery 任务协同处理，重新分配前必须再次进行预览容量准入。

### Compatibility and operations

- 旧 Engine 未上报容量、指标过期或配置非法时，节点不得参与新预览调度，不能将未知容量视为无限容量。
- 管理端能够查看最大预览数、当前预览数、待确认预留数、指标时间和拒绝调度原因。
- 出厂容量通过 Engine 环境变量部署；Go 管理端只展示上报值，不提供会与设备出厂规格冲突的独立容量推导。

## Acceptance Criteria

- [ ] Engine 可通过 `NIKO_ENGINE_MAX_PREVIEW_STREAMS` 配置最大并发预览数，并上报容量和当前使用量。
- [ ] Create、EnablePlayback、DisablePlayback、Destroy 和失败回滚均能幂等维护 `preview_in_use`。
- [ ] Go 调度严格满足 `preview_in_use + pending < preview_capacity`，满载时拒绝新预览。
- [ ] 多节点选择按调度后占用率最低、节点 ID 稳定打破平局。
- [ ] 已有推理 Pipeline 追加预览占一个名额，重复请求同一预览不重复计数。
- [ ] `MediaStream.node_id` 在启动成功后持久化，后续控制命令始终路由到同一节点。
- [ ] 并发请求不会超分最后一个预览名额。
- [ ] 旧 Engine、非法配置和过期指标不会进入新预览调度池。
- [ ] Engine 配置/计数测试、Go 调度测试、多节点并发测试和迁移测试通过。

## Dependencies

- 依赖 `07-12-edge-node-task-routing` 提供所有流控制命令的显式 `node_id` 契约。
- 与 `07-12-edge-node-liveness-recovery` 协同定义节点离线后的预览恢复。
- 管理端展示由 `07-12-edge-node-contract-security` 集成。

## Out Of Scope

- 根据分辨率、码率、MPP decoder/encoder 数量、网络带宽或 NPU 使用率动态推导最大预览数。
- 在 Go 控制面实现 Rockchip、Ascend、CUDA 或 macOS 的硬件容量探测。
- 重写预览 Pipeline 或跨节点故障恢复状态机。

## Notes

- 最大预览数是设备型号的出厂规格；若未来需要自动检测，只能作为生产测试/设备初始化工具，不应在运行时覆盖显式环境变量。
