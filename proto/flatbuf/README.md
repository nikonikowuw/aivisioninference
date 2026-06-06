# FlatBuffers IPC 协议定义

Go 控制面与 C++ 数据面之间的进程间通信协议。

## 文件结构

```
proto/flatbuf/
├── README.md           # 本文件
├── common.fbs          # 共享基础类型 (坐标、枚举、几何)
├── envelope.fbs        # IPC 信封 (版本、消息类型、路由)
├── commands.fbs        # Go → C++ 控制指令
├── results.fbs         # C++ → Go 结果/状态上报
├── CHANGELOG.md        # 版本变更日志
└── COMPATIBILITY.md    # 兼容性矩阵
```

## 消息架构

```
┌─────────────────────────────────────────────────────────┐
│                    IPCEnvelope                           │
│  ┌───────────────────────────────────────────────────┐  │
│  │ schema_version: ushort                            │  │
│  │ message_type:   MessageType                       │  │
│  │ sequence_id:    uint                              │  │
│  │ timestamp_ns:   ulong                             │  │
│  │ payload:        MessagePayload (union)            │  │
│  │   ├─ StartStreamCmd      (Go → C++)               │  │
│  │   ├─ StopStreamCmd       (Go → C++)               │  │
│  │   ├─ UpdateConfigCmd     (Go → C++)               │  │
│  │   ├─ LoadAlgoCmd         (Go → C++)               │  │
│  │   ├─ UnloadAlgoCmd       (Go → C++)               │  │
│  │   ├─ InferenceResultMsg  (C++ → Go)               │  │
│  │   ├─ StreamStatusMsg     (C++ → Go)               │  │
│  │   ├─ AlgoLoadResultMsg   (C++ → Go)               │  │
│  │   ├─ EngineMetricsMsg    (C++ → Go)               │  │
│  │   ├─ WorkerStatusMsg     (C++ → Go)               │  │
│  │   ├─ HeartbeatCmd        (双向)                   │  │
│  │   └─ HeartbeatAckMsg     (双向)                   │  │
│  └───────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────┘
```

## 编译

### 生成 Go 代码

```bash
flatbuf_dir="proto/flatbuf"
out_dir="app/internal/pkg/ipc/flatbuf"

# 生成 Go 代码
flatc --go \
  --gen-all \
  --go-namespace "flatbuf" \
  -o "$out_dir" \
  "$flatbuf_dir/envelope.fbs"
```

### 生成 C++ 代码

```bash
flatbuf_dir="proto/flatbuf"
out_dir="engine/include/ipc"

# 生成 C++ 头文件
flatc --cpp \
  --gen-all \
  --cpp-std C++17 \
  -o "$out_dir" \
  "$flatbuf_dir/envelope.fbs"
```

### 兼容性检查

```bash
# 检查新 schema 是否向后兼容 (需要 flatbuffers 23.5.26+)
flatc --conform "$flatbuf_dir/envelope.fbs" \
  --conform-check "$flatbuf_dir/envelope_prev.fbs"
```

## Go Model → FlatBuffers 映射

| Go Model | FlatBuffers Table | 转换层 |
|----------|------------------|--------|
| `InferTask` | `StartStreamCmd` | Go IPC 发送端 |
| `InferTaskAlgorithm` | `AlgoBinding` (在 StartStreamCmd 中) | Go IPC 发送端 |
| `InferTaskAlgorithm.AIParams` | `AlgoBinding.algo_params_json` | JSON 透传 |
| `InferTaskAlgorithm.ROIRegions` | `AlgoBinding.roi_regions` | JSON → FlatBuffers |
| `AlgorithmPackage` | `LoadAlgoCmd` | Go IPC 发送端 |
| `SmartRecord` | `InferenceResultMsg` (接收后映射) | Go IPC 接收端 |
| `InferTask.Status` | `StreamStatusMsg.status` | Go IPC 接收端 |

## 隐含任务: Go 侧 JSON → FlatBuffers 转换层

Go 侧的 ROI/MARK/LINE 区域存储为 `datatypes.JSON` (JSONB),
需要转换为 FlatBuffers 的 `[Polygon]` 结构。

建议新增 `app/internal/pkg/ipc/converter.go`:

```go
package ipc

// JSONToPolygons 将 JSONB 格式的区域数据转换为 FlatBuffers Polygon 列表
// 输入: [{"points": [{"x": 100, "y": 200}, ...]}, ...]
// 输出: FlatBuffers 序列化的 [Polygon]
func JSONToPolygons(jsonData datatypes.JSON) []byte { ... }

// PolygonsToJSON 将 FlatBuffers Polygon 列表转换为 JSONB
// 用于 C++ 上报结果时的反向转换
func PolygonsToJSON(data []byte) datatypes.JSON { ... }
```

## 版本演进

详见 [CHANGELOG.md](CHANGELOG.md) 和 [COMPATIBILITY.md](COMPATIBILITY.md)。

### 快速参考

```
版本号 = major * 100 + minor

新增可选字段 → minor++ (完全兼容)
结构性变更   → major++ (需要过渡版本)
```

### 字段废弃规则

1. **永远不删除字段** — 只标记 deprecated
2. **保留字段 ID 占位** — 删除会导致 ID 错位
3. **至少保留 2 个大版本** — 确保滚动升级完成
