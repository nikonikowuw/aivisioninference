# Backend Error Handling

## API Error Shape

HTTP APIs use `internal/pkg/response` for the public response shape:

- Success: `response.OK(c, data)` -> `{ "code": "OK", "message": "success", "data": ... }`
- Pagination: `response.Page(c, list, total, page, pageSize)` -> `data.list`, `data.total`, `data.page`, `data.page_size`
- Failure: `response.Err(c, err)` -> business error code (string, e.g. `"REQ_BAD_REQUEST"`) and localized/default message

All error codes are `string` constants in `SCREAMING_SNAKE_CASE` with module prefixes (`AUTH_*`, `REQ_*`, `NOT_FOUND_*`, etc.). See `app/internal/pkg/errors/errors.go` for the full mapping.

Reference files:
- `app/internal/pkg/response/response.go`
- `app/internal/pkg/errors/errors.go`
- `app/internal/handler/edge_node.go`

## Business Errors

Use `internal/pkg/errors.AppError` for expected domain failures. Add new codes in `app/internal/pkg/errors/errors.go` and make sure translations exist for supported frontend languages.

Examples:
- `ErrEdgeNodeNameTaken` for duplicate edge node names.
- `ErrNotFound` when a service maps `gorm.ErrRecordNotFound`.
- `ErrForbidden` when active tasks prevent an edge node delete.

Do not return raw database, SDK, panic, or internal command text directly to the frontend.

## Wrapping Internal Failures

Wrap unexpected internal failures with `%w` so logs and callers can inspect the chain:

```go
return nil, "", fmt.Errorf("生成节点JWT令牌失败: %w", err)
```

Expected business errors should remain `apperrors.New(...)` so `response.Err` can map them to the standard JSON and HTTP status.

## Handler Pattern

Handlers should:

1. Bind request data with `ShouldBindJSON`, `ShouldBindQuery`, or path params.
2. Convert binding errors through the local bad-request helper.
3. Call the service with `c.Request.Context()`.
4. Return through `response.OK`, `response.Page`, or `response.Err`.

`app/internal/handler/edge_node.go` is the reference shape.

## Boundary Exceptions

Middleware such as Auth, RBAC, RateLimit, CORS, ZLM webhook, and edge-node auth may abort the request at the protocol boundary. Their responses still need to preserve either the project response format or the third-party protocol contract.

Asynq task handlers return errors to Asynq and log operational context. See `app/internal/task/handler.go` for JSON payload parsing and task-level return errors.

## Common Mistakes

- Using `errors.New("some user-facing sentence")` in a service that is returned to an API client.
- Returning GORM `ErrRecordNotFound` from a service instead of mapping it to an app error.
- Swallowing an error with `_` when a storage, DB, MQTT, or token operation fails.
- Translating frontend-visible text in a repository. Translation belongs at API/error presentation boundaries.
