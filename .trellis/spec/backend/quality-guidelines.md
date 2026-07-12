# Backend Quality Guidelines

## Before Editing

Read the relevant `.rules/golang-pro.md` section before Go changes. Use CodeGraph first when locating symbols in this repository because `.codegraph/` is present.

## Context And Lifecycle

Every service and repository method involved in a request should receive `context.Context` first. Goroutines need an explicit lifecycle. Long-running services such as `SystemService` keep a cancel function and expose `Close()`.

Reference files:
- `app/internal/service/system.go`
- `app/internal/service/edge_node.go`

## API And Swagger

New or changed HTTP handlers need:

- DTO structs with validation tags.
- Swagger annotations beside the handler.
- Route registration under `/api/v1` unless it is a documented health/static/webhook exception.
- Uniform `response` package output.

Run `cd app && make swag` after annotation changes.

## Dependency Graph

When adding backend objects:

- Repository/service/handler providers go through `app/internal/router/deps.go` and `wire.go`.
- App-level dependencies go through `app/internal/server/`.
- Regenerate with `cd app && make wire`.

Do not edit `wire_gen.go` manually.

## Tests

Use focused tests for shared utilities, protocol matching, converters, repositories with meaningful query logic, and services with state transitions. Existing examples include:

- `app/internal/pkg/mqttmux/mux_test.go` for MQTT topic matching.
- `app/internal/pkg/controlproto/` tests for protocol parsing/conversion.
- `app/internal/service/*_test.go` for service-level behavior where present.

Run a focused package test first, then `cd app && make unit-test` for broader validation when the change affects shared behavior.

## Notifier Provider Pattern

Use an interface + registry pattern when implementing multiple notification channels:

```go
// Define the interface in the service package
type Notifier interface {
    Send(ctx context.Context, event AlertEvent) error
}

// Registry holds all configured providers
type NotifierRegistry struct {
    providers map[string]Notifier
}

func NewNotifierRegistry(cfg *NotifierConfig) *NotifierRegistry {
    r := &NotifierRegistry{providers: make(map[string]Notifier)}
    if cfg.WebhookURL != "" {
        r.providers["webhook"] = NewWebhookNotifier(cfg.WebhookURL)
    }
    if cfg.SMTPHost != "" {
        r.providers["email"] = NewEmailNotifier(cfg.SMTPConfig)
    }
    // Additional providers registered by name
    return r
}

func (r *NotifierRegistry) Send(ctx context.Context, channels []string, event AlertEvent) error {
    for _, ch := range channels {
        if notifier, ok := r.providers[ch]; ok {
            _ = notifier.Send(ctx, event)
        }
    }
    return nil
}
```

**Key Rules:**
- Register providers once at service construction in `deps.go`, not per-request.
- Notifier configuration (webhook URLs, SMTP settings) comes from `config` structs populated by env/config files, not from database settings.
- Each `alert_rule` selects channels via its `notify_channels` JSONB field (e.g., `["webhook","email"]`).
- Notification failures must not prevent alert event creation. Log the error and continue.
- The registry is injected via Wire into the alert engine service.

## Alert Engine Inline Evaluation

The alert engine runs inline after heartbeat metrics persistence (not as a separate async worker):

```go
func (s *EdgeNodeService) HandleHeartbeat(ctx context.Context, id string, req *dto.HeartbeatRequest) error {
    // ... existing heartbeat processing ...

    // Persist metrics
    record := s.buildMetricsRecord(id, req)
    if err := s.metricsRepo.Create(ctx, record); err != nil { ... }

    // Evaluate alert rules inline (non-fatal on error)
    if err := s.alertEngine.EvaluateAfterHeartbeat(ctx, id, record); err != nil {
        log.Warn("alert evaluation failed", ...)
    }
    return nil
}
```

**Key Rules:**
- Alert evaluation must be non-blocking on failure — a failing rule evaluator should not prevent heartbeat processing.
- For large numbers of nodes (>1000), consider moving evaluation to an async job queue.
- Silence tracking uses an in-memory `sync.Map` keyed by `(rule_id, node_id)`. On restart, silence state is reset (acceptable for edge cases).
- Node offline/online transitions (MQTT LWT, reconnection) trigger dedicated evaluator methods (`EvaluateNodeOffline`, `EvaluateNodeBackOnline`).

## Backend Anti-Patterns

- Adding business rules to repositories.
- Calling external systems from handlers instead of services.
- Skipping Wire because a direct constructor call is faster to write.
- Adding a model without a migration entry and necessary indexes.
- Using raw SQL with concatenated user input.
- Returning Chinese or English display strings directly from low-level packages when an error code should be translated.
