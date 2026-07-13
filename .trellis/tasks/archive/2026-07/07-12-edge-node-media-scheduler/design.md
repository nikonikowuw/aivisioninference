# 预览节点容量调度器设计

## 设计目标

将预览容量抽象为每节点的最大并发预览数。Engine/HAL 负责加载设备出厂容量并统计当前预览数，Go 负责容量预留、节点选择和稳定路由。

## Engine 侧

### 配置

Engine 读取：

```text
NIKO_ENGINE_MAX_PREVIEW_STREAMS=<positive integer>
```

该值由设备型号出厂压力测试确定。环境变量缺失、为零或无法解析时，容量状态标记为无效，节点不得接收新的自动预览分配。

### 计数语义

`preview_in_use` 按已启用预览输出的唯一 Pipeline 计数：

- 新建 preview Pipeline：`+1`。
- 已有 inference Pipeline 调用 EnablePlayback：`+1`。
- 对已启用 playback 的 Pipeline 重复调用：`+0`。
- DisablePlayback：`-1`。
- Destroy 带 playback 的 Pipeline：`-1`。
- 创建或启用过程中失败：不得留下计数。

计数更新与 Pipeline playback 状态在同一锁和状态转换内完成，避免指标与真实运行状态漂移。

### 上报契约

心跳或 Engine 指标追加：

```text
preview_capacity
preview_in_use
preview_capacity_valid
timestamp_ns
```

平台 HAL 只负责实现预览链路。Rockchip 内部可以使用 MPP，其他平台使用对应硬件媒体实现；统一指标不暴露底层资源细节。

## Go 侧

### 快照

Go 按 `node_id` 保存最新预览容量快照和接收时间。以下情况不可调度：

- 节点离线或禁用；
- `preview_capacity_valid=false`；
- `preview_capacity <= 0`；
- 指标超过 TTL；
- `preview_in_use > preview_capacity`，表示状态异常。

### 需求与预留

新预览需求固定为一个名额。同一 `stream_key` 已在同节点启用预览时需求为零并复用原分配。

调度事务读取有效的预览快照和未完成预留，并验证：

```text
preview_in_use + pending_reservations + demand <= preview_capacity
```

预留以 `stream_key` 唯一，包含节点、状态和租约过期时间。启动成功确认预留；启动失败或停止后释放；过期 pending 由清理流程回收。

### 节点排序

候选分数：

```text
score = (preview_in_use + pending_reservations + demand) / preview_capacity
```

选择 score 最低者，分数相同按节点 ID 升序。没有候选时返回结构化原因：`offline`、`disabled`、`unconfigured`、`stale_metrics` 或 `preview_full`。

## 路由生命周期

1. 根据设备/流生成稳定 `stream_key`。
2. 若已有活动 `MediaStream.node_id`，先验证并复用原节点。
3. 否则在事务中选择节点并创建一个预览名额的 pending 预留。
4. 使用显式 `node_id` 发送 Start/StartPlayback。
5. 成功后写入 `MediaStream.node_id` 并确认预留；失败则释放预留。
6. Stop、Status、重试和恢复从 `MediaStream.node_id` 读取节点，不重新选择当前最低负载节点。

## 兼容与回滚

- 协议字段只追加；旧 Engine 的零值按无效容量处理。
- Go 不提供独立的“最大预览数”覆盖值，避免管理端与设备出厂规格产生双重事实来源。
- 可通过关闭自动调度停止新分配；已有流仍按持久化 node ID 停止和查询。
- 回滚代码时保留 `MediaStream.node_id` 和预留表，避免路由信息丢失。

## 已替代方案

不再使用解码槽、编码槽、出口带宽或分辨率权重构造通用媒体资源向量。它们超出当前“只调度预览数”的产品范围，也会把平台硬件细节泄漏到 Go 控制面。
