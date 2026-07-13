# 预览节点容量调度器实施计划

## 前置门禁

- [ ] 与 `07-12-edge-node-task-routing` 对齐显式 `node_id` 的 EngineClient/MQTT 契约。
- [ ] 确认容量唯一来源为 Engine 环境变量和上报值，Go 不独立配置或推导最大预览数。
- [ ] 复核现有媒体资源向量实现，列出需要删除、迁移或兼容保留的数据库与协议字段。

## 实施顺序

1. [ ] 在 Engine 配置中加入 `NIKO_ENGINE_MAX_PREVIEW_STREAMS`，实现正整数校验和无效配置日志。
2. [ ] 在 PipelineManager 中按 playback enabled 状态幂等维护 `preview_in_use`，覆盖 Create、EnablePlayback、DisablePlayback、Destroy 和失败回滚。
3. [ ] 扩展心跳或 Engine 指标协议，上报 `preview_capacity`、`preview_in_use`、有效标记和采集时间；同步生成 Go/C++ 协议代码。
4. [ ] Go 按节点缓存/持久化最新预览快照和 TTL，旧 Engine 或非法容量判定为不可调度。
5. [ ] 将调度器从解码/编码/带宽资源向量收敛为单一预览名额，删除无关硬约束和拒绝原因。
6. [ ] 将数据库预留收敛为每个 stream 一个预览名额，保持租约、确认、释放和并发锁能力。
7. [ ] 实现候选过滤与排序：`in_use + pending + demand <= capacity`，按调度后占用率和节点 ID 排序。
8. [ ] 接入 StreamManager/MediaService：已有同流预览复用，否则预留节点并使用显式 node ID 启动。
9. [ ] 启动成功后持久化 `MediaStream.node_id` 并确认预留；失败、停止和超时路径释放预留。
10. [ ] 让 Stop、Status、重试和服务重启恢复全部从 `MediaStream.node_id` 路由。
11. [ ] 与节点离线恢复任务集成重新分配流程；本任务只提供重新准入能力。
12. [ ] 清理旧的解码槽、编码槽、出口带宽 API/模型/测试，必要时提供一次性兼容迁移。

## 测试矩阵

- Engine 环境变量：缺失、非法、零、正整数。
- Engine 计数：新预览、推理追加预览、重复 EnablePlayback、DisablePlayback、Destroy、启动失败。
- Go 调度：空闲、满载、多节点占用率、稳定 tie-break、过期指标、旧 Engine。
- 并发预留：多个请求争抢最后一个名额时最多一个成功。
- 生命周期：Start 成功确认、Start 失败释放、Stop 释放、租约超时回收、服务重启恢复。
- 路由：Start/Stop/Status 始终使用持久化 node ID。

## 验证命令

```bash
cd app && go test ./internal/pkg/controlproto ./internal/repository ./internal/service ./internal/handler ./internal/task
cd app && make build
cd engine && cmake --build build -j && ctest --test-dir build --output-on-failure
```

## 风险与回滚点

- 现有实现已引入媒体资源向量字段和协议；收敛时必须检查数据库迁移兼容，不能直接删除生产列导致回滚失败。
- `preview_in_use` 必须来自真实 Pipeline 状态，不可仅靠命令次数累加。
- Go 预留只覆盖指标上报延迟，不能永久替代 Engine 真实计数；pending 必须有租约回收。
- 不手工修改 FlatBuffers 生成文件；修改 schema 后重新生成双端代码。

## 完成门禁

- [ ] 使用 `trellis-check` 完成 Engine、Go、协议、并发和生命周期检查。
- [ ] 更新相关 spec，记录“最大预览数由 Engine 环境变量配置并上报”的稳定契约。
- [ ] 与 `07-12-edge-node-task-routing` 和 liveness/recovery 任务完成跨任务集成验收。
