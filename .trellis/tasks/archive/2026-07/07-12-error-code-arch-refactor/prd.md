# 错误码架构重构：数字码改为字母码

## Goal

将系统错误码从数字格式（如 10001, 20001）改为字母格式（如 `REQ_BAD_REQUEST`, `AUTH_UNAUTHORIZED`），统一与 engine 层的字母码格式。涉及 Go 控制面、前端、i18n 三端改动。

## Background

当前系统存在两套错误码体系：

- **Go 业务层**使用数字码（`int`），通过 `code / 10000` 的隐式段映射决定 HTTP 状态码
- **Engine 层**已使用字母码（`string`），如 `ENGINE_TIMEOUT`、`NO_FACE`、`DB_WRITE_ERROR`

每次 engine 返回字母码时，Go 业务层需要手工映射到数字码，存在遗漏风险。此外现有 10xxx 数字段已被六类不同性质的错误抢占（License、Person、Device、Engine、System、Permission），段分类已失去意义。

将两套体系统一为字母码，消除翻译层，提升可读性和维护性。

## Requirements

### 功能需求 (FR)

- **FR-1**: 所有 Go 业务层错误码常量值从 `int` 改为 `string`（SCREAMING_SNAKE_CASE + 模块前缀）
- **FR-2**: `AppError.Code` 字段类型从 `int` 改为 `string`
- **FR-3**: API JSON 响应中 `code` 字段类型从 `number` 改为 `string`（字段名不变）
- **FR-4**: `codeToHTTPStatus()` 从隐式数字段除法改为 `map[string]int` 显式映射
- **FR-5**: i18n 翻译存储从 `map[int]string` 改为 `map[string]string`
- **FR-6**: 前端 `ApiError.code` 和 `ApiResponse.code` 从 `number` 改为 `string`
- **FR-7**: 前端 `isUnauthenticated()` 从数字范围检查改为字符串集合检查
- **FR-8**: 成功码从 `0` 改为 `"OK"`

### 非功能需求 (NFR)

- **NFR-1**: 业务代码中 `apperrors.New(ErrXxx, "")` 调用处不需要改代码（常量名不变，仅类型自动跟随）
- **NFR-2**: Engine 层 FlatBuffers schema 和 DB 中已存储的字母码无需迁移
- **NFR-3**: 前后端必须同步上线（API response 类型变更）
- **NFR-4**: 日志中的错误码从 `zap.Int` 改为 `zap.String`

### 约束

- 不修改 Engine/C++ 的 FlatBuffers schema
- 不修改 DB schema（`embedding_error_code` 已是 `VARCHAR`）
- 不修改 `wire_gen.go`（无新增依赖注入）

## Acceptance Criteria

- [ ] **AC-1**: `go build ./...` 编译通过，无类型错误
- [ ] **AC-2**: `go test ./internal/...` 全部通过
- [ ] **AC-3**: TypeScript `tsc --noEmit` 无类型错误
- [ ] **AC-4**: `npx vitest run` 全部通过
- [ ] **AC-5**: API 返回 JSON 中 `code` 字段为字符串而非数字
- [ ] **AC-6**: 触发未登录错误时，前端收到 code 为 `"AUTH_UNAUTHORIZED"`，翻译显示正确，HTTP 状态为 401
- [ ] **AC-7**: 触发参数错误时，前端收到 code 为 `"REQ_BAD_REQUEST"`，翻译显示正确，HTTP 状态为 400
- [ ] **AC-8**: 触发资源不存在时，前端收到 code 为 `"NOT_FOUND_DEVICE"` 等，HTTP 状态为 404
- [ ] **AC-9**: 前端 `isUnauthenticated()` 正确识别所有 `AUTH_*` 错误
- [ ] **AC-10**: 所有 6 个语言（zh-CN, zh-TW, en-US, id-ID, ja-JP, ko-KR）的错误码翻译均正确对应新字母码

## Not Doing

- 不修改 Engine/C++ 的 FlatBuffers schema
- 不修改 DB schema
- 不修改 `algorithms/` 目录下的代码
- 不涉及 ZLMediaKit 配置
