// Package errors provides unified business error codes and error types
// for the niko-admin application.
package errors

import (
	"fmt"
	"strings"

	"github.com/niko-admin/niko-admin/internal/pkg/i18n"
)

// Standard business error codes.
const (
	Success = "OK"

	// REQ — 通用请求校验 (原 1xxxx 通用参数).
	ErrBadRequest            = "REQ_BAD_REQUEST"
	ErrCannotDisableSelf     = "REQ_CANNOT_DISABLE_SELF"
	ErrStartTimeFormat       = "REQ_START_TIME_FORMAT"
	ErrEndTimeFormat         = "REQ_END_TIME_FORMAT"
	ErrTimeRangeOrder        = "REQ_TIME_RANGE_ORDER"
	ErrFileTooLarge          = "REQ_FILE_TOO_LARGE"
	ErrFileInvalidType       = "REQ_FILE_INVALID_TYPE"
	ErrNotImplemented        = "REQ_NOT_IMPLEMENTED"
	ErrTooManyRequests       = "REQ_TOO_MANY_REQUESTS"

	// USER — 用户管理.
	ErrEmailTaken            = "USER_EMAIL_TAKEN"
	ErrOldPasswordWrong      = "USER_OLD_PASSWORD_WRONG"
	ErrHierarchyLevelUser    = "USER_HIERARCHY_LEVEL"
	ErrMailNotEnabled        = "USER_MAIL_NOT_ENABLED"
	ErrSMTPTestFailed        = "USER_SMTP_TEST_FAILED"
	ErrIMAPTestFailed        = "USER_IMAP_TEST_FAILED"
	ErrTokenInvalidOrExpired = "USER_TOKEN_INVALID"
	ErrUsernameTaken         = "USER_USERNAME_TAKEN"
	ErrCannotDeleteSelf      = "USER_CANNOT_DELETE_SELF"
	ErrCannotResetSelf       = "USER_CANNOT_RESET_SELF"

	// ROLE — 角色管理.
	ErrHierarchyLevelRole    = "ROLE_HIERARCHY_LEVEL"
	ErrRoleNameTaken         = "ROLE_NAME_TAKEN"
	ErrRoleLevelInvalid      = "ROLE_LEVEL_INVALID"

	// CSV — CSV 导入.
	ErrCSVInvalidContent     = "CSV_INVALID_CONTENT"
	ErrCSVRowLimitExceeded   = "CSV_ROW_LIMIT_EXCEEDED"
	ErrCSVColumnRequired     = "CSV_COLUMN_REQUIRED"
	ErrCSVStatusInvalid      = "CSV_STATUS_INVALID"
	ErrCSVHeaderInvalid      = "CSV_HEADER_INVALID"
	ErrCSVDuplicateUsername  = "CSV_DUPLICATE_USERNAME"
	ErrCSVWeakPassword       = "CSV_WEAK_PASSWORD"
	ErrCSVInvalidEmail       = "CSV_INVALID_EMAIL"

	// PERSON — 人员管理.
	ErrPersonImageRequired  = "PERSON_IMAGE_REQUIRED"  // 请上传人脸图片
	ErrPersonImageDuplicate = "PERSON_IMAGE_DUPLICATE" // 图片已存在
	ErrPersonCodeDuplicate  = "PERSON_CODE_DUPLICATE"  // 人员编号已存在
	ErrPersonStatusNoRetry  = "PERSON_STATUS_NO_RETRY" // 当前状态无需重提
	ErrArchiveRequired      = "PERSON_ARCHIVE_REQUIRED" // 请上传压缩包
	ErrArchiveUnsupported   = "PERSON_ARCHIVE_UNSUPPORTED" // 不支持的压缩包格式

	// Face search errors.
	ErrFaceExtractFailed  = "PERSON_FACE_EXTRACT_FAILED"   // 人脸特征提取失败
	ErrFaceSearchNoResult = "PERSON_FACE_SEARCH_NO_RESULT" // 未找到相似人员

	// PERM — 权限管理.
	ErrPermissionCodeTaken = "PERM_CODE_TAKEN"
	ErrPermissionAssigned  = "PERM_ASSIGNED"
	ErrRoleAssigned        = "PERM_ROLE_ASSIGNED"

	// FILE — 文件/上传.
	ErrFileContentIncomplete = "FILE_CONTENT_INCOMPLETE"
	ErrChunkCountMismatch    = "FILE_CHUNK_COUNT_MISMATCH"
	ErrInvalidChunkIndex     = "FILE_INVALID_CHUNK_INDEX"
	ErrChunkIncomplete       = "FILE_CHUNK_INCOMPLETE"
	ErrUploadCanceled        = "FILE_UPLOAD_CANCELED"
	ErrFileSizeMismatch      = "FILE_SIZE_MISMATCH"
	ErrFileChecksumMismatch  = "FILE_CHECKSUM_MISMATCH"

	// SYS — 系统管理.
	ErrCleanupRunning        = "SYS_CLEANUP_RUNNING"
	ErrCleanupDisabled       = "SYS_CLEANUP_DISABLED"
	ErrTimeSyncFailed        = "SYS_TIME_SYNC_FAILED"
	ErrInvalidTimezone       = "SYS_INVALID_TIMEZONE"
	ErrTimezoneFileNotFound  = "SYS_TIMEZONE_FILE_NOT_FOUND"
	ErrSetTimeFailed         = "SYS_SET_TIME_FAILED"
	ErrTimeOutOfRange        = "SYS_TIME_OUT_OF_RANGE"
	ErrNetworkConfigFailed   = "SYS_NETWORK_CONFIG_FAILED"
	ErrNetworkRollbackFailed = "SYS_NETWORK_ROLLBACK_FAILED"
	ErrNetworkConfirmFailed  = "SYS_NETWORK_CONFIRM_FAILED"
	ErrWebhookPushFailed     = "SYS_WEBHOOK_PUSH_FAILED"

	// LICENSE — 授权管理.
	ErrLicenseInvalid        = "LICENSE_INVALID"          // 授权文件无效或签名验证失败
	ErrLicenseDeviceMismatch = "LICENSE_DEVICE_MISMATCH"  // 授权文件绑定的设备指纹与当前设备不匹配
	ErrLicenseNotAuthorized  = "LICENSE_NOT_AUTHORIZED"   // 算法未在授权范围内
	ErrLicenseExpired        = "LICENSE_EXPIRED"          // 授权已过期
	ErrLicenseNotYetValid    = "LICENSE_NOT_YET_VALID"    // 授权尚未生效
	ErrLicenseStreamLimit    = "LICENSE_STREAM_LIMIT"     // 授权并发路数不足
	ErrLicenseNotConfigured  = "LICENSE_NOT_CONFIGURED"   // 授权公钥未配置

	// ENGINE — 引擎通信.
	ErrEngineNotReady    = "ENGINE_NOT_READY"      // 推理引擎未就绪
	ErrConnectionTimeout = "ENGINE_CONNECTION_TIMEOUT" // 连接超时
	ErrEngineResponseBad = "ENGINE_RESPONSE_BAD"   // 引擎响应异常
	ErrConnectionFailed  = "ENGINE_CONNECTION_FAILED"  // 连接失败
	ErrConnectionTestOK  = "ENGINE_CONNECTION_TEST_OK" // 连接测试成功，流可达

	// DEV — 设备管理.
	ErrDeviceNameTaken              = "DEV_NAME_TAKEN"
	ErrRTSPURLRequired              = "DEV_RTSP_URL_REQUIRED"
	ErrGB28181CodeRequired          = "DEV_GB28181_CODE_REQUIRED"
	ErrDeviceGroupNotEmpty          = "DEV_GROUP_NOT_EMPTY"
	ErrResourceConflict             = "DEV_RESOURCE_CONFLICT"
	ErrInvalidGB28181DeviceCode     = "DEV_GB28181_CODE_INVALID"
	ErrGB28181DeviceOffline         = "DEV_GB28181_OFFLINE"
	ErrInvalidPlaybackAction        = "DEV_INVALID_PLAYBACK_ACTION"
	ErrAlarmDeviceRequired          = "DEV_ALARM_DEVICE_REQUIRED"
	ErrTimeWindowFormat             = "DEV_TIME_WINDOW_FORMAT"
	ErrTaskStatusNotCancelable      = "DEV_TASK_STATUS_NOT_CANCELABLE"
	ErrDeviceExternalKeyTaken       = "DEV_EXTERNAL_KEY_TAKEN"
	ErrDeviceDisabled               = "DEV_DISABLED"
	ErrDeviceOffline                = "DEV_OFFLINE"
	ErrDeviceTypeInvalid            = "DEV_TYPE_INVALID"
	CodeVersionIncompatible         = "DEV_VERSION_INCOMPATIBLE"

	// NODE — 边缘节点.
	ErrEdgeNodeNameTaken            = "NODE_NAME_TAKEN"
	ErrEdgeNodeNotFound             = "NODE_NOT_FOUND"
	ErrEdgeNodeOffline              = "NODE_OFFLINE"
	ErrEdgeNodeDisabled             = "NODE_DISABLED"
	ErrEdgeNodeFull                 = "NODE_FULL"
	ErrEdgeNodeAlgorithmUnavailable = "NODE_ALGORITHM_UNAVAILABLE"

	// AUTH — 身份认证 (原 2xxxx).
	ErrUnauthorized       = "AUTH_UNAUTHORIZED"
	ErrTokenExpired       = "AUTH_TOKEN_EXPIRED"
	ErrTokenInvalid       = "AUTH_TOKEN_INVALID"
	ErrRefreshTokenReuse  = "AUTH_TOKEN_REUSED"
	ErrInvalidCredentials = "AUTH_INVALID_CREDENTIALS"

	// FORBIDDEN — 权限/ACL (原 3xxxx).
	ErrForbidden        = "FORBIDDEN"
	ErrOriginNotAllowed = "FORBIDDEN_ORIGIN"
	ErrUserDisabled     = "FORBIDDEN_USER_DISABLED"

	// NOT_FOUND — 资源不存在 (原 4xxxx).
	ErrNotFound               = "NOT_FOUND"
	ErrFeedbackNotFound       = "NOT_FOUND_FEEDBACK"
	ErrUserNotFound           = "NOT_FOUND_USER"
	ErrRoleNotFound           = "NOT_FOUND_ROLE"
	ErrPermissionNotFound     = "NOT_FOUND_PERMISSION"
	ErrTaskNotFound           = "NOT_FOUND_TASK"
	ErrDeviceNotFound         = "NOT_FOUND_DEVICE"
	ErrFileNotFound           = "NOT_FOUND_FILE"
	ErrUploadSessionNotFound  = "NOT_FOUND_UPLOAD_SESSION"
	ErrDeviceGroupNotFound    = "NOT_FOUND_DEVICE_GROUP"
	ErrLicenseNotFound        = "NOT_FOUND_LICENSE"
	ErrAITimeScheduleNotFound = "NOT_FOUND_TIME_SCHEDULE"

	// INTERNAL — 服务器内部 (原 5xxxx).
	ErrInternal            = "INTERNAL_ERROR"
	ErrStreamStateNotFound = "INTERNAL_STREAM_STATE_NOT_FOUND"
	ErrZLMNotConfigured    = "INTERNAL_ZLM_NOT_CONFIGURED"
	ErrFingerprintExtract  = "INTERNAL_FINGERPRINT_EXTRACT_FAIL"
)

const defaultLanguage = "en"

// AppError represents a business-level error with a code and message.
type AppError struct {
	Code    string `json:"code"`
	Message string `json:"message"`

	// localizedMessage 标记 Message 是否已经按当前请求语言翻译，可直接返回给前端。
	localizedMessage bool
}

// Error implements the error interface.
func (e *AppError) Error() string {
	return fmt.Sprintf("code=%s, message=%s", e.Code, e.Message)
}

func New(code string, msg string) *AppError {
	if msg == "" {
		return &AppError{Code: code, Message: DefaultMessage(code, defaultLanguage), localizedMessage: false}
	}
	return &AppError{Code: code, Message: msg, localizedMessage: false}
}

// IsLocalizedMessage reports whether the message was already translated for the current request.
func (e *AppError) IsLocalizedMessage() bool {
	return e != nil && e.localizedMessage
}

// NewLocalized creates an AppError whose explicit message is safe to return to the frontend.
func NewLocalized(code string, msg string) *AppError {
	if msg == "" {
		return New(code, msg)
	}
	return &AppError{Code: code, Message: msg, localizedMessage: true}
}

// Newf creates a new AppError with a formatted explicit message.
func Newf(code string, format string, args ...interface{}) *AppError {
	return &AppError{Code: code, Message: fmt.Sprintf(format, args...), localizedMessage: true}
}

// DefaultMessage returns the default message for a business error code in the specified language.
func DefaultMessage(code string, lang string) string {
	msg := i18n.Translate(lang, code)
	if strings.HasPrefix(msg, "unknown error (code=") {
		return "Unknown error"
	}
	return msg
}
