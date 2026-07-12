# Backend Logging Guidelines

## Logger

Use Zap (`zap.L()` or an injected `*zap.Logger`) for application logs. Avoid `fmt.Println` and `log.Println` in long-lived server, service, task, or MQTT code. CLI migration code may use standard logging for process-fatal startup/migration messages, as shown in `app/cmd/migrate/main.go`.

Reference files:
- `app/internal/service/system.go` logs metrics loop startup and shutdown with `zap.L().Info`.
- `app/internal/task/handler.go` logs Asynq task payload context.
- `app/internal/pkg/response/response.go` logs unexpected non-`AppError` responses.

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

## Hot Paths

Metrics, heartbeat, MQTT message dispatch, and inference result ingestion can be high volume. Avoid per-frame or per-message info logs in hot paths unless they are sampled, aggregated, or behind a debug setting. `SystemService.startMetricsLoop` caches and records metrics without logging every collection tick.

## Error Level Use

- `Debug`: verbose protocol details during local diagnosis.
- `Info`: lifecycle events, task completion, important state transitions.
- `Warn`: recoverable degradation, retries, invalid optional input.
- `Error`: failed external calls, persistence failures, unexpected internal errors.

Do not log and then return the same error at every layer. Add context at the boundary that owns the operation.
