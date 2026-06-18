# FlatBuffers 消息 Schema 兼容性矩阵

当前传输主链路为 MQTT/HTTP；本矩阵只约束 FlatBuffers 消息 schema 的版本兼容性。

## 版本号规则

```
schema_version = major * 100 + minor

major: 结构性变更 (可能不兼容)
minor: 新增可选字段 (完全兼容)
```

**当前版本**: 100 (v1.0)

---

## 兼容性矩阵

```
            │ C++ v1.0 │ C++ v1.1 │ C++ v2.0 │ C++ v3.0 │
────────────┼──────────┼──────────┼──────────┼──────────┤
Go  v1.0    │    ✅    │    ✅*   │    ✅*   │    ❌    │
Go  v1.1    │    ✅*   │    ✅    │    ✅*   │    ❌    │
Go  v2.0    │    ✅*   │    ✅*   │    ✅    │    ✅*   │
Go  v3.0    │    ❌    │    ❌    │    ✅*   │    ✅    │

✅  = 完全兼容
✅* = 功能降级但不崩溃 (新字段使用默认值)
❌  = 不兼容, 必须同步升级
```

---

## 兼容性检查规则 (C++ 侧实现参考)

```cpp
// 伪代码: C++ 接收消息时的版本检查
void onMessageReceived(const ControlEnvelope* envelope) {
    uint16_t received = envelope->schema_version();
    uint16_t recv_major = received / 100;
    uint16_t recv_minor = received % 100;

    // 主版本不兼容: 对方太旧
    if (recv_major < MY_MAJOR - 1) {
        // 差距超过 1 个大版本, 拒绝通信
        sendError(VersionMismatch, "peer protocol too old");
        return;
    }

    // 主版本不兼容: 对方太新
    if (recv_major > MY_MAJOR) {
        // 我太旧了, 但可以尝试读取已知字段
        log.Warn("peer speaks v%d, I'm v%d. Some fields ignored.",
                 recv_major, MY_MAJOR);
    }

    // 次版本差异: 安全忽略未知字段
    if (recv_minor > MY_MINOR) {
        log.Debug("peer has minor version %d, I have %d.", recv_minor, MY_MINOR);
    }

    // 正常处理
    dispatch(envelope);
}
```

---

## Go 侧实现参考

```go
// 伪代码: Go 接收消息时的版本检查
func (r *Receiver) onMessage(envelope *controlproto.ControlEnvelope) error {
    received := envelope.SchemaVersion()
    recvMajor := received / 100
    recvMinor := received % 100

    // 主版本不兼容: 对方太旧
    if recvMajor < myMajor-1 {
        return ErrVersionMismatch.WithDetail("peer protocol too old")
    }

    // 主版本不兼容: 对方太新
    if recvMajor > myMajor {
        log.Warn("peer speaks newer protocol",
            zap.Uint16("peer_major", recvMajor),
            zap.Uint16("my_major", myMajor))
    }

    // 正常处理 (FlatBuffers 自动忽略未知字段)
    return r.dispatch(envelope)
}
```

---

## 各版本的版本号范围

| 版本 | schema_version | major | minor | 状态 |
|------|---------------|-------|-------|------|
| v1.0 | 100 | 1 | 0 | 当前 |
| v1.1 | 101 | 1 | 1 | 规划中 |
| v2.0 | 200 | 2 | 0 | 远期 |
| v3.0 | 300 | 3 | 0 | 远期 |

---

## 字段废弃流程

当需要废弃某个字段时:

1. **不删除字段** — FlatBuffers 按字段 ID 读取, 删除会导致 ID 错位
2. **标记 deprecated** — 在 schema 注释中说明, 在 CHANGELOG 中记录
3. **停止写入** — 新版本不再填充该字段
4. **保留读取** — 旧版本写入的值仍然可读
5. **至少保留 2 个大版本** — 确保滚动升级完成后再考虑清理

示例:

```fbs
table InferenceResultMsg {
    // ... 其他字段 ...

    // deprecated since v2.0, use tracks instead
    // 保留位置, 不可删除
    detections:[BoundingBox];

    // v2.0 新增
    tracks:[TrackResult];
    has_tracks:bool = false;
}
```
