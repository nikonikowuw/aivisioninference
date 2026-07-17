# Backend Logging Guidelines

## Logger

Use Zap (`zap.L()` or an injected `*zap.Logger`) for application logs. Avoid `fmt.Println` and `log.Println` in long-lived server, service, task, or MQTT code. CLI migration code may use standard logging for process-fatal startup/migration messages, as shown in `app/cmd/migrate/main.go`.

For request-scoped logging that carries trace context, use `log.Ctx(ctx)` instead of `zap.L()`:

```go
// ✅ 正确: 携带全链路追踪上下文
log.Ctx(ctx).Info("device created", zap.String("device_id", id))

// ❌ 错误: 丢失 trace_id / user_id 上下文
zap.L().Info("device created")
```

### 全链路追踪上下文

每条日志自动注入 `trace_id`、`user_id`、`tenant_id`、`span_id` 字段，通过 `context.Context` 传递。

- **TraceID 中间件** (`middleware.TraceID`) 在每个请求入口生成/透传 Trace ID，注入 `LogContext`
- **`log.Ctx(ctx)`** 从 context 提取 `LogContext` 并作为结构化字段追加到日志条目
- **`logger.Ctx(ctx)`** 在已初始化的 Logger 实例上也可用

### 敏感数据脱敏

日志系统通过 `SafeCore` 自动脱敏敏感字段。默认脱敏 key 列表：

```
password, secret, token, access_token, refresh_token, private_key, authorization, cookie
```

- 通过 `log.sanitize.enabled` 控制（默认关闭）
- 可通过 `log.sanitize.keys` 自定义脱敏 key 列表
- 所有日志输出（文件 + stdout）统一脱敏

### 缓冲写入

文件日志已启用 `BufferedWriteSyncer` 提升写入性能：

| 日志类型 | 缓冲区大小 | 刷新间隔 |
|----------|-----------|---------|
| app      | 256KB     | 5s      |
| error    | 64KB      | 5s      |
| access   | 0（关闭）  | —       |

配置项：`log.<type>.buffer_size`、`log.<type>.flush_interval`。

### 运行时动态日志级别

通过 `PUT /api/v1/system/log/level` 修改日志级别（需 admin/superadmin 权限）。

- 支持: `debug`, `info`, `warn`, `error`, `dpanic`, `panic`, `fatal`
- 修改即时生效，无需重启服务
- `error` 级别的日志不受运行时调级影响（始终使用独立 Error 级别 core）

## What To Include

Log operational context that helps reconstruct distributed flows:

- `node_id`, `task_id`, `stream_id`, `algo_package_id`
- MQTT topic, command type, trace/sequence id when available
- Counts and status values for batch operations
- Duration and retry count for external operations

Prefer structured fields:

```go
zap.L().Info("imap sync completed", zap.Int("synced", count))
```

## What Not To Log

Never log secrets or credential material:

- JWT access/refresh tokens
- Node tokens
- MQTT passwords
- Database passwords
- Object storage credentials and presigned URLs with sensitive query strings
- Full identity documents or biometric payloads

For edge-node heartbeat, algorithm deployment, and storage operations, log identifiers and statuses rather than full request bodies.

即使 SafeCore 已启用，也不应依赖脱敏机制来记录明文凭证——防御纵深而非安全底线。

## Hot Paths

Metrics, heartbeat, MQTT message dispatch, and inference result ingestion can be high volume. Avoid per-frame or per-message info logs in hot paths unless they are sampled, aggregated, or behind a debug setting. `SystemService.startMetricsLoop` caches and records metrics without logging every collection tick.

对于高频日志路径可启用采样（`log.sampling.enabled`），通过 `zapcore.NewSamplerWithOptions` 控制采样频率。

## Error Level Use

- `Debug`: verbose protocol details during local diagnosis.
- `Info`: lifecycle events, task completion, important state transitions.
- `Warn`: recoverable degradation, retries, invalid optional input.
- `Error`: failed external calls, persistence failures, unexpected internal errors.

Do not log and then return the same error at every layer. Add context at the boundary that owns the operation.

## 告警集成

当 `log.error_hook.enabled` 启用时，Warn/Error 级别的日志事件会通过 ErrorHook 回调发送。

- 支持 Webhook URL（`log.error_hook.url`），异步 POST JSON 到配置地址
- 可通过 `log.error_hook.min_level` 调整触发级别（warn / error / dpanic）
- 告警回调在 Write 路径中同步调用，不阻塞日志写入

## Reference Files

核心实现:
- `app/internal/pkg/log/log.go` — Logger 初始化、Ctx()、BuildCore
- `app/internal/pkg/log/context.go` — LogContext 类型与上下文辅助函数
- `app/internal/pkg/log/safe_core.go` — SafeCore 脱敏包装器
- `app/internal/pkg/log/error_hook.go` — ErrorHook 接口与 WebhookErrorHook
- `app/internal/middleware/trace.go` — TraceID 中间件
- `app/internal/handler/log.go` — 动态日志级别 API handler

关键调用点:
- `app/internal/service/system.go` — 指标循环启停日志
- `app/internal/task/handler.go` — Asynq 任务上下文日志
- `app/internal/pkg/response/response.go` — 非 AppError 响应日志
