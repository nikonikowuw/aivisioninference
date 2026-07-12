# FlatBuffers 消息 Schema 变更日志

所有版本的结构性变更和兼容性说明。当前传输主链路为 MQTT/HTTP；本文件只描述 FlatBuffers 消息 schema 的演进。

---

## v1.2 (schema_version = 102) — 媒体容量指标

**发布日期**: 2026-07-12

### 新增字段

`EngineMetricsMsg` 追加解码/编码会话、槽位使用量、出口带宽、preview/inference/mixed Pipeline 数、`media_metrics_valid`，以及预览准入字段 `preview_capacity`、`preview_in_use`、`preview_capacity_valid`。预览容量来自 Engine 的 `NIKO_ENGINE_MAX_PREVIEW_STREAMS`，使用量按已启用 playback 的唯一 Pipeline 统计。

### 兼容性

- 旧读取方自动忽略新增字段。
- 新读取方读取旧 Engine 消息时得到零值且 `preview_capacity_valid=false`，预览自动调度安全降级为不可用。

---

## v1.0 (schema_version = 100) — 初始版本

**发布日期**: TBD

### 新增消息类型

| 方向   | 类型                               | 说明               |
| ------ | ---------------------------------- | ------------------ |
| Go→C++ | `StartStreamCmd`                   | 启动视频流推理     |
| Go→C++ | `StopStreamCmd`                    | 停止视频流推理     |
| Go→C++ | `UpdateConfigCmd`                  | 运行时更新算法配置 |
| Go→C++ | `LoadAlgoCmd`                      | 加载/切换算法库    |
| Go→C++ | `UnloadAlgoCmd`                    | 卸载算法库         |
| C++→Go | `InferenceResultMsg`               | 推理结果           |
| C++→Go | `StreamStatusMsg`                  | 流状态变更         |
| C++→Go | `AlgoLoadResultMsg`                | 算法加载结果       |
| C++→Go | `EngineMetricsMsg`                 | 引擎运行指标       |
| 双向   | `HeartbeatCmd` / `HeartbeatAckMsg` | 心跳探活           |

### 兼容性

- 初始版本, 无兼容性约束

---

## v1.1 (schema_version = 101) — 性能观测增强

**发布日期**: TBD (与 v1.0 同期, 视实现情况决定是否拆分)

### 新增字段

| 消息                 | 新增字段              | 类型   | 默认值 | 说明         |
| -------------------- | --------------------- | ------ | ------ | ------------ |
| `BoundingBox`        | `label_name`          | string | `""`   | 人可读标签名 |
| `BoundingBox`        | `track_id`            | int    | `-1`   | 目标跟踪 ID  |
| `InferenceResultMsg` | `infer_time_us`       | uint   | `0`    | 推理耗时     |
| `InferenceResultMsg` | `preprocess_time_us`  | uint   | `0`    | 预处理耗时   |
| `InferenceResultMsg` | `postprocess_time_us` | uint   | `0`    | 后处理耗时   |

### 新增消息类型

| 方向   | 类型              | 说明               |
| ------ | ----------------- | ------------------ |
| C++→Go | `WorkerStatusMsg` | 单 Worker 状态诊断 |

### 兼容性

| 读取方   | 写入方   | 结果                |
| -------- | -------- | ------------------- |
| Go v1.0  | C++ v1.1 | ✅ 新字段自动忽略   |
| Go v1.1  | C++ v1.0 | ✅ 新字段读到默认值 |
| C++ v1.0 | Go v1.1  | ✅ 新字段自动忽略   |
| C++ v1.1 | Go v1.0  | ✅ 新字段自动忽略   |

---

## v2.0 (schema_version = 200) — 坐标系升级 (规划中)

**发布日期**: TBD

### 变更内容

| 消息                 | 变更                                      | 说明           |
| -------------------- | ----------------------------------------- | -------------- |
| `Polygon`            | 新增 `points_normalized` + `coord_system` | 支持归一化坐标 |
| `InferenceResultMsg` | 新增 `background_path`                    | 背景图路径     |

### 兼容性

| 读取方   | 写入方   | 结果                                         |
| -------- | -------- | -------------------------------------------- |
| Go v1.x  | C++ v2.0 | ✅ 新字段忽略, 旧 `points` 仍有效            |
| Go v2.0  | C++ v1.x | ✅ `points_normalized` 为空, 回退到 `points` |
| C++ v1.x | Go v2.0  | ✅ 新字段忽略                                |
| C++ v2.0 | Go v1.x  | ✅ 新字段忽略                                |

### 迁移策略

1. v2.0 发布后, C++ 侧**双写** `points` 和 `points_normalized`
2. Go 侧优先读取 `points_normalized`, 回退到 `points`
3. v3.0 时废弃 `points` 字段 (标记 deprecated, 不删除)

---

## v3.0 (schema_version = 300) — 推理结果结构化 (远期规划)

**发布日期**: TBD

### 变更内容

计划将 `InferenceResultMsg.detections` (扁平列表) 重构为按目标跟踪分组的结构:

```fbs
// 新增
table TrackResult {
  track_id:int;
  detections:[BoundingBox];
  trajectory:[PointF];
}

// InferenceResultMsg 新增
tracks:[TrackResult];       // 分组结果
has_tracks:bool = false;    // 标记是否使用新模式
```

### 迁移策略

- v2.x → v3.0 需要双写过渡期
- 详见发布时的迁移指南
