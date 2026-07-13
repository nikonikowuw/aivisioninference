# 错误码架构重构实施计划

## 前置评审门禁

- [ ] 确认设计文档中的全量错误码命名映射表，特别是模块前缀分配
- [ ] 确认 `codeToHTTPStatus()` 改用 `map[string]int` 显式映射
- [ ] 确认前端 `isUnauthenticated()` 改用字符串集合检查
- [ ] 确认 i18n 翻译存储改 `map[string]string` 后不影响现有功能

## 实施顺序

### 阶段 1：Go 核心类型层

1. [ ] **修改 `app/internal/pkg/errors/errors.go`**
   - `Code` 从 `int` 改为 `string`
   - 所有常量值从数字改为字母串（按设计文档映射表）
   - `Success` 改为 `"OK"`
   - `AppError.Error()` 的格式化从 `code=%d` 改为 `code=%s`
   - `DefaultMessage()` 签名从 `Translate(lang, code int)` 改为 `Translate(lang, code string)`

2. [ ] **修改 `app/internal/pkg/errors/errors_test.go`**
   - 更新测试用例，传字符串而非数字

3. [ ] **修改 `app/internal/pkg/i18n/i18n.go`**
   - `storage` 类型从 `map[string]map[int]string` 改为 `map[string]map[string]string`
   - 所有语言翻译的 key 从数字改为字母串
   - `Translate` 函数签名改为 `func Translate(lang string, code string) string`
   - `Translate()` 的回退消息从 `"unknown error (code=%d)"` 改为 `"unknown error (code=%s)"`
   - `RegisterMessages` 签名的 `map[int]string` 改为 `map[string]string`

### 阶段 2：Go 中间件与响应层

4. [ ] **修改 `app/internal/pkg/response/response.go`**
   - `Response.Code` 从 `int` 改 `string`，JSON tag `json:"code"` 不变
   - `codeToHTTPStatus()` 从 `switch code / 10000` 改为 `map[string]int` 显式查找
   - `contextLanguage()` 不变（语言检测逻辑无关）

5. [ ] **修改 `app/internal/pkg/response/response_test.go`**
   - 更新测试断言，code 类型改为 string

6. [ ] **修改 `app/internal/middleware/error.go`**
   - `logAppError` 中的 `appErr.Code` 日志字段从 `zap.Int` 改为 `zap.String`
   - `appErr.Code >= 50000` 的范围判断改为字符串匹配（`strings.HasPrefix(appErr.Code, "INTERNAL")`）

### 阶段 3：Web 前端

7. [ ] **修改 `web/src/services/api.ts`**
   - `ApiResponse.code`：`number` → `string`
   - `ApiError.code`：`number` → `string`
   - `getErrorMessage(code: number)` → `getErrorMessage(code: string)`
   - `isUnauthenticated()`：`this.code >= 20001 && this.code <= 20004` → 字符串集合检查
   - `resolveApiErrorMessage()` 签名更新
   - 构造 `ApiError` 时传字符串 code

8. [ ] **修改 `web/src/services/api.test.ts`**
   - 测试中的 `99999` 改为 `"UNKNOWN"` 等

### 阶段 4：前端 i18n 本地化

9. [ ] **修改所有 locale 文件 `web/src/locales/*/common.ts`**
   - `en-US/common.ts`：`message.error` 的 key 从 `"10001"` → `"REQ_BAD_REQUEST"` 等
   - `zh-CN/common.ts`：同上
   - `zh-TW/common.ts`：同上
   - `id-ID/common.ts`：同上
   - `ja-JP/common.ts`：同上
   - `ko-KR/common.ts`：同上

### 阶段 5：验证

10. [ ] **Go 构建验证**
    ```bash
    cd app && go build ./...
    ```

11. [ ] **Go 单元测试**
    ```bash
    cd app && go test ./internal/pkg/errors/... ./internal/pkg/response/... ./internal/pkg/i18n/... ./internal/middleware/...
    ```

12. [ ] **Go 全量测试**
    ```bash
    cd app && go test ./internal/...
    ```

13. [ ] **前端构建**
    ```bash
    cd web && npx tsc --noEmit && npx vite build
    ```

14. [ ] **前端测试**
    ```bash
    cd web && npx vitest run
    ```

## 测试矩阵

| 测试场景 | 验证点 | 涉及层 |
|---|---|---|
| API 响应 code 字段类型 | 返回 `"REQ_BAD_REQUEST"` 而非 `10001` | Go 响应层 |
| HTTP 状态码映射 | `FORBIDDEN_*` → 403, `NOT_FOUND_*` → 404 | response.go |
| 前端错误消息翻译 | code `"AUTH_UNAUTHORIZED"` 显示「未登录」 | 前端 i18n |
| 401 自动跳转 | `AUTH_*` 类错误触发重新登录 | 前端 api.ts |
| 服务端日志 | 日志中用字符串 code 而非数字 | middleware |
| i18n fallback | 未知 code 显示「Unknown error」 | i18n.go |
| 成功响应 | `"OK"` 而非 `0` | response.go |

## 兼容与回滚

- API JSON 的 `code` 字段名不变，仅类型从 `number` 变 `string`
- 前端所有新旧部署不兼容，必须前后端同步上线
- FlatBuffers schema 和 DB 中的 `embedding_error_code` 已经存储字符串，无需迁移
- Engine 层字母码不涉及改动
- 回滚时需同时恢复前后端

## 风险

- 前端 `isUnauthenticated()` 的字符串集合若遗漏新 AUTH 错误，会导致已登录用户不会被踢回登录页
- `codeToHTTPStatus()` 的 map 若遗漏某个错误码，会返回零值 HTTP 200，导致前端捕获不到错误
- 所有 locale 文件的 key 必须保持严格一致性，漏改一个语言会导致该语言翻译失效
- `zap.Int("code", ...)` 改为 `zap.String("code", ...)` 可能影响日志解析工具

## 完成门禁

- [ ] `go build ./...` 通过
- [ ] `go test ./internal/...` 全部通过
- [ ] `npx tsc --noEmit` 无类型错误
- [ ] `npx vitest run` 全部通过
- [ ] 前端手动测试：触发若干常见错误（未登录、参数错误、资源不存在），确认显示正确翻译和 HTTP 状态码
- [ ] 使用 `trellis-check` 完成规格与质量检查
