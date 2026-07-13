# 错误码架构重构设计

## 现状分析

### 当前双轨制

| 层 | 格式 | 例子 |
|---|---|---|
| Go 业务层 | 数字 int | `10001` (ErrBadRequest), `50001` (ErrInternal) |
| Engine/IPC 层 | 字母 string | `ENGINE_TIMEOUT`, `NO_FACE`, `DB_WRITE_ERROR` |
| DB (embedding_error_code) | 字母 string | `"NO_FACE"` |

当前 engine 返回字母码时，Go 业务层需要手工映射到数字码才能统一，存在遗漏风险。

### 数字码段已混乱

现有 10xxx 段被六类不同性质的错误抢占：

| 码段 | 归类 | 问题 |
|---|---|---|
| 10001-10029 | 通用参数、用户、角色 | 杂糅 |
| 10030-10042 | 系统管理 | 和 10001 同段 |
| 10050-10056 | 授权 License | 和系统管理冲突 |
| 10060-10067 | 人员管理 | 和授权冲突 |
| 10070-10079 | 权限/文件上传 | 和人员冲突 |
| 10080-10084 | 引擎连接 | 和文件上传冲突 |
| 10100-10121 | 设备管理 | 和引擎连接不同类 |

段分类已经失去意义。

## 设计目标

1. **统一格式**：Go 业务层 + Engine 层统一使用字母码，消除翻译层
2. **模块可辨**：通过前缀命名空间区分错误来源模块
3. **向后兼容**：API JSON 的 `code` 字段名不变，仅类型从 `int` 变 `string`
4. **最小改动**：业务代码中 `apperrors.New(ErrXxx, "")` 调用处不需要改代码（常量名不变）

## 命名规范

### 通约

```
{UPPER_CASE_MODULE_PREFIX}_{SCREAMING_SNAKE_CASE_NAME}
```

### 模块前缀

| 前缀 | 对应原码段 | 说明 |
|---|---|---|
| `AUTH_*` | 2xxxx | 身份认证 |
| `FORBIDDEN_*` | 3xxxx | 权限/ACL |
| `NOT_FOUND_*` | 4xxxx | 资源不存在 |
| `INTERNAL_*` | 5xxxx | 服务器内部 |
| `REQ_*` | 1xxxx (通用参数校验) | 请求参数校验 |
| `USER_*` | 用户管理相关 | 用户增删改 |
| `ROLE_*` | 角色管理相关 | 角色 |
| `PERM_*` | 权限管理相关 | 权限 |
| `FILE_*` | 文件/上传相关 | 文件上传、分片 |
| `CSV_*` | CSV 导入相关 | CSV 导入校验 |
| `SYS_*` | 系统运维相关 | NTP、网络、Webhook |
| `LICENSE_*` | 授权管理 | License 校验 |
| `PERSON_*` | 人员管理 | 人脸、特征 |
| `ENGINE_*` | 引擎通信 | 引擎连接测试 |
| `DEV_*` | 设备管理 | 设备 CRUD |
| `NODE_*` | 边缘节点 | 节点管理 |

### 成功码

```
OK
```

## 全量错误码映射表

### AUTH (原 2xxxx)

```go
ErrUnauthorized            = "AUTH_UNAUTHORIZED"          // 20001
ErrTokenExpired            = "AUTH_TOKEN_EXPIRED"         // 20002
ErrTokenInvalid            = "AUTH_TOKEN_INVALID"         // 20003
ErrRefreshTokenReuse       = "AUTH_TOKEN_REUSED"          // 20004
ErrInvalidCredentials      = "AUTH_INVALID_CREDENTIALS"   // 20005
```

### FORBIDDEN (原 3xxxx)

```go
ErrForbidden               = "FORBIDDEN"                  // 30001
ErrOriginNotAllowed        = "FORBIDDEN_ORIGIN"           // 30002
ErrUserDisabled            = "FORBIDDEN_USER_DISABLED"    // 30003
```

### NOT_FOUND (原 4xxxx)

```go
ErrNotFound                = "NOT_FOUND"                  // 40001
ErrFeedbackNotFound        = "NOT_FOUND_FEEDBACK"         // 40002
ErrUserNotFound            = "NOT_FOUND_USER"             // 40003
ErrRoleNotFound            = "NOT_FOUND_ROLE"             // 40004
ErrPermissionNotFound      = "NOT_FOUND_PERMISSION"       // 40005
ErrTaskNotFound            = "NOT_FOUND_TASK"             // 40006
ErrDeviceNotFound          = "NOT_FOUND_DEVICE"           // 40007
ErrFileNotFound            = "NOT_FOUND_FILE"             // 40008
ErrUploadSessionNotFound   = "NOT_FOUND_UPLOAD_SESSION"   // 40009
ErrDeviceGroupNotFound     = "NOT_FOUND_DEVICE_GROUP"     // 40010
ErrLicenseNotFound         = "NOT_FOUND_LICENSE"          // 40011
ErrAITimeScheduleNotFound  = "NOT_FOUND_TIME_SCHEDULE"    // 40012
```

### INTERNAL (原 5xxxx)

```go
ErrInternal                = "INTERNAL_ERROR"                     // 50001
ErrStreamStateNotFound     = "INTERNAL_STREAM_STATE_NOT_FOUND"    // 50002
ErrZLMNotConfigured        = "INTERNAL_ZLM_NOT_CONFIGURED"        // 50003
ErrFingerprintExtract      = "INTERNAL_FINGERPRINT_EXTRACT_FAIL"  // 50004
```

### REQ — 通用请求校验 (原 1xxxx 通用参数)

```go
ErrBadRequest              = "REQ_BAD_REQUEST"             // 10001
ErrCannotDisableSelf       = "REQ_CANNOT_DISABLE_SELF"    // 10002
ErrStartTimeFormat         = "REQ_START_TIME_FORMAT"       // 10008
ErrEndTimeFormat           = "REQ_END_TIME_FORMAT"         // 10009
ErrTimeRangeOrder          = "REQ_TIME_RANGE_ORDER"        // 10010
ErrFileTooLarge            = "REQ_FILE_TOO_LARGE"          // 10011
ErrFileInvalidType         = "REQ_FILE_INVALID_TYPE"       // 10012
ErrNotImplemented          = "REQ_NOT_IMPLEMENTED"         // 10041
ErrTooManyRequests         = "REQ_TOO_MANY_REQUESTS"       // 10042
```

### USER — 用户管理

```go
ErrEmailTaken              = "USER_EMAIL_TAKEN"            // 10005
ErrOldPasswordWrong        = "USER_OLD_PASSWORD_WRONG"    // 10006
ErrMailNotEnabled          = "USER_MAIL_NOT_ENABLED"       // 10013
ErrSMTPTestFailed          = "USER_SMTP_TEST_FAILED"       // 10014
ErrIMAPTestFailed          = "USER_IMAP_TEST_FAILED"       // 10015
ErrTokenInvalidOrExpired   = "USER_TOKEN_INVALID"          // 10016
ErrUsernameTaken           = "USER_USERNAME_TAKEN"         // 10025
ErrCannotDeleteSelf        = "USER_CANNOT_DELETE_SELF"     // 10026
ErrCannotResetSelf         = "USER_CANNOT_RESET_SELF"      // 10027
ErrHierarchyLevelUser      = "USER_HIERARCHY_LEVEL"        // 10003
```

### ROLE — 角色管理

```go
ErrHierarchyLevelRole      = "ROLE_HIERARCHY_LEVEL"        // 10004
ErrRoleNameTaken           = "ROLE_NAME_TAKEN"             // 10028
ErrRoleLevelInvalid        = "ROLE_LEVEL_INVALID"          // 10029
```

### CSV — CSV 导入

```go
ErrCSVInvalidContent       = "CSV_INVALID_CONTENT"         // 10017
ErrCSVRowLimitExceeded     = "CSV_ROW_LIMIT_EXCEEDED"      // 10018
ErrCSVColumnRequired       = "CSV_COLUMN_REQUIRED"         // 10019
ErrCSVStatusInvalid        = "CSV_STATUS_INVALID"          // 10020
ErrCSVHeaderInvalid        = "CSV_HEADER_INVALID"          // 10021
ErrCSVDuplicateUsername    = "CSV_DUPLICATE_USERNAME"      // 10022
ErrCSVWeakPassword         = "CSV_WEAK_PASSWORD"           // 10023
ErrCSVInvalidEmail         = "CSV_INVALID_EMAIL"           // 10024
```

### PERM — 权限管理

```go
ErrPermissionCodeTaken     = "PERM_CODE_TAKEN"             // 10070
ErrPermissionAssigned      = "PERM_ASSIGNED"               // 10071
ErrRoleAssigned            = "PERM_ROLE_ASSIGNED"          // 10072
```

### FILE — 文件/上传

```go
ErrFileContentIncomplete   = "FILE_CONTENT_INCOMPLETE"     // 10073
ErrChunkCountMismatch      = "FILE_CHUNK_COUNT_MISMATCH"   // 10074
ErrInvalidChunkIndex       = "FILE_INVALID_CHUNK_INDEX"    // 10075
ErrChunkIncomplete         = "FILE_CHUNK_INCOMPLETE"       // 10076
ErrUploadCanceled          = "FILE_UPLOAD_CANCELED"        // 10077
ErrFileSizeMismatch        = "FILE_SIZE_MISMATCH"          // 10078
ErrFileChecksumMismatch    = "FILE_CHECKSUM_MISMATCH"      // 10079
```

### SYS — 系统管理

```go
ErrCleanupRunning          = "SYS_CLEANUP_RUNNING"         // 10030
ErrCleanupDisabled         = "SYS_CLEANUP_DISABLED"        // 10031
ErrTimeSyncFailed          = "SYS_TIME_SYNC_FAILED"        // 10032
ErrInvalidTimezone         = "SYS_INVALID_TIMEZONE"        // 10033
ErrTimezoneFileNotFound    = "SYS_TIMEZONE_FILE_NOT_FOUND" // 10034
ErrSetTimeFailed           = "SYS_SET_TIME_FAILED"         // 10035
ErrTimeOutOfRange          = "SYS_TIME_OUT_OF_RANGE"       // 10036
ErrNetworkConfigFailed     = "SYS_NETWORK_CONFIG_FAILED"   // 10037
ErrNetworkRollbackFailed   = "SYS_NETWORK_ROLLBACK_FAILED" // 10038
ErrNetworkConfirmFailed    = "SYS_NETWORK_CONFIRM_FAILED"  // 10039
ErrWebhookPushFailed       = "SYS_WEBHOOK_PUSH_FAILED"     // 10040
```

### LICENSE — 授权管理

```go
ErrLicenseInvalid          = "LICENSE_INVALID"             // 10050
ErrLicenseDeviceMismatch   = "LICENSE_DEVICE_MISMATCH"     // 10051
ErrLicenseNotAuthorized    = "LICENSE_NOT_AUTHORIZED"      // 10052
ErrLicenseExpired          = "LICENSE_EXPIRED"             // 10053
ErrLicenseNotYetValid      = "LICENSE_NOT_YET_VALID"       // 10054
ErrLicenseStreamLimit      = "LICENSE_STREAM_LIMIT"        // 10055
ErrLicenseNotConfigured    = "LICENSE_NOT_CONFIGURED"      // 10056
```

### PERSON — 人员管理

```go
ErrPersonImageRequired     = "PERSON_IMAGE_REQUIRED"       // 10060
ErrPersonImageDuplicate    = "PERSON_IMAGE_DUPLICATE"      // 10061
ErrPersonCodeDuplicate     = "PERSON_CODE_DUPLICATE"       // 10062
ErrPersonStatusNoRetry     = "PERSON_STATUS_NO_RETRY"      // 10063
ErrArchiveRequired         = "PERSON_ARCHIVE_REQUIRED"     // 10064
ErrArchiveUnsupported      = "PERSON_ARCHIVE_UNSUPPORTED"  // 10065
ErrFaceExtractFailed       = "PERSON_FACE_EXTRACT_FAILED"  // 10066
ErrFaceSearchNoResult      = "PERSON_FACE_SEARCH_NO_RESULT" // 10067
```

### ENGINE — 引擎通信

```go
ErrEngineNotReady          = "ENGINE_NOT_READY"            // 10080
ErrConnectionTimeout       = "ENGINE_CONNECTION_TIMEOUT"   // 10081
ErrEngineResponseBad       = "ENGINE_RESPONSE_BAD"         // 10082
ErrConnectionFailed        = "ENGINE_CONNECTION_FAILED"    // 10083
ErrConnectionTestOK        = "ENGINE_CONNECTION_TEST_OK"   // 10084
```

### DEV — 设备管理

```go
ErrDeviceNameTaken         = "DEV_NAME_TAKEN"              // 10100
ErrRTSPURLRequired         = "DEV_RTSP_URL_REQUIRED"       // 10101
ErrGB28181CodeRequired     = "DEV_GB28181_CODE_REQUIRED"   // 10102
ErrDeviceGroupNotEmpty     = "DEV_GROUP_NOT_EMPTY"         // 10103
ErrResourceConflict        = "DEV_RESOURCE_CONFLICT"       // 10104
ErrInvalidGB28181DeviceCode = "DEV_GB28181_CODE_INVALID"   // 10105
ErrGB28181DeviceOffline    = "DEV_GB28181_OFFLINE"         // 10106
ErrInvalidPlaybackAction   = "DEV_INVALID_PLAYBACK_ACTION" // 10107
ErrAlarmDeviceRequired     = "DEV_ALARM_DEVICE_REQUIRED"   // 10108
ErrTimeWindowFormat        = "DEV_TIME_WINDOW_FORMAT"      // 10109
ErrTaskStatusNotCancelable = "DEV_TASK_STATUS_NOT_CANCELABLE" // 10110
ErrDeviceExternalKeyTaken  = "DEV_EXTERNAL_KEY_TAKEN"      // 10111
ErrDeviceDisabled          = "DEV_DISABLED"                // 10112
ErrDeviceOffline           = "DEV_OFFLINE"                 // 10113
ErrDeviceTypeInvalid       = "DEV_TYPE_INVALID"            // 10114
CodeVersionIncompatible    = "DEV_VERSION_INCOMPATIBLE"    // 10115
```

### NODE — 边缘节点

```go
ErrEdgeNodeNameTaken               = "NODE_NAME_TAKEN"               // 10116
ErrEdgeNodeNotFound                = "NODE_NOT_FOUND"                // 10117
ErrEdgeNodeOffline                 = "NODE_OFFLINE"                  // 10118
ErrEdgeNodeDisabled                = "NODE_DISABLED"                 // 10119
ErrEdgeNodeFull                    = "NODE_FULL"                     // 10120
ErrEdgeNodeAlgorithmUnavailable    = "NODE_ALGORITHM_UNAVAILABLE"    // 10121
```

## 涉及修改的核心文件

| 文件 | 改动内容 |
|---|---|
| `app/internal/pkg/errors/errors.go` | `Code int` → `Code string`；所有常量值改为字母串 |
| `app/internal/pkg/errors/errors_test.go` | 测试断言更新 |
| `app/internal/pkg/response/response.go` | `Response.Code` 从 `int` 改 `string`；`codeToHTTPStatus()` 改为 `map[string]int` |
| `app/internal/pkg/response/response_test.go` | 测试断言更新 |
| `app/internal/pkg/i18n/i18n.go` | i18n `storage` 从 `map[int]string` 改 `map[string]string` |
| `app/internal/middleware/error.go` | `appErr.Code` 的比较逻辑从数字范围改为字符串匹配 |
| `web/src/services/api.ts` | `ApiResponse.code` / `ApiError.code` 从 `number` 改 `string`；`isUnauthenticated()` 改为字符串集合检查 |
| `web/src/services/api.test.ts` | 测试更新 |
| `web/src/locales/*/common.ts` | 所有错误码 key 从 `"10001"` 改为 `"REQ_BAD_REQUEST"` |
| `app/internal/service/*.go` | 调用处无需改（常量名不变，类型自动跟随） |

## 不涉及改动的文件

- 所有 `apperrors.New(ErrXxx, "")` 调用：常量名不变，类型自动跟随
- FlatBuffers schema：Engine 层已用字母码
- DB schema：`embedding_error_code` 已存储字母码，无需迁移
