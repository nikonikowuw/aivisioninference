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
	Success = 0

	// Client errors (1xxxx).
	ErrBadRequest            = 10001
	ErrCannotDisableSelf     = 10002
	ErrHierarchyLevelUser    = 10003
	ErrHierarchyLevelRole    = 10004
	ErrEmailTaken            = 10005
	ErrOldPasswordWrong      = 10006
	ErrStartTimeFormat       = 10008
	ErrEndTimeFormat         = 10009
	ErrTimeRangeOrder        = 10010
	ErrFileTooLarge          = 10011
	ErrFileInvalidType       = 10012
	ErrMailNotEnabled        = 10013
	ErrSMTPTestFailed        = 10014
	ErrIMAPTestFailed        = 10015
	ErrTokenInvalidOrExpired = 10016
	ErrCSVInvalidContent     = 10017
	ErrCSVRowLimitExceeded   = 10018
	ErrCSVColumnRequired     = 10019
	ErrCSVStatusInvalid      = 10020
	ErrCSVHeaderInvalid      = 10021
	ErrCSVDuplicateUsername  = 10022
	ErrCSVWeakPassword       = 10023
	ErrCSVInvalidEmail       = 10024
	ErrUsernameTaken         = 10025
	ErrCannotDeleteSelf      = 10026
	ErrCannotResetSelf       = 10027
	ErrRoleNameTaken         = 10028
	ErrRoleLevelInvalid      = 10029

	// Person management errors (1006x).
	ErrPersonImageRequired  = 10060 // 请上传人脸图片
	ErrPersonImageDuplicate = 10061 // 图片已存在
	ErrPersonCodeDuplicate  = 10062 // 人员编号已存在
	ErrPersonStatusNoRetry  = 10063 // 当前状态无需重提
	ErrArchiveRequired      = 10064 // 请上传压缩包
	ErrArchiveUnsupported   = 10065 // 不支持的压缩包格式

	// Domain validation errors (1007x-101xx).
	ErrPermissionCodeTaken      = 10070
	ErrPermissionAssigned       = 10071
	ErrRoleAssigned             = 10072
	ErrFileContentIncomplete    = 10073
	ErrChunkCountMismatch       = 10074
	ErrInvalidChunkIndex        = 10075
	ErrChunkIncomplete          = 10076
	ErrUploadCanceled           = 10077
	ErrFileSizeMismatch         = 10078
	ErrFileChecksumMismatch     = 10079
	ErrDeviceNameTaken          = 10100
	ErrRTSPURLRequired          = 10101
	ErrGB28181CodeRequired      = 10102
	ErrDeviceGroupNotEmpty      = 10103
	ErrResourceConflict         = 10104
	ErrInvalidGB28181DeviceCode = 10105
	ErrGB28181DeviceOffline     = 10106
	ErrInvalidPlaybackAction    = 10107
	ErrAlarmDeviceRequired      = 10108
	ErrTimeWindowFormat         = 10109
	ErrTaskStatusNotCancelable  = 10110
	ErrDeviceExternalKeyTaken   = 10111

	// Auth errors (2xxxx).
	ErrUnauthorized       = 20001
	ErrTokenExpired       = 20002
	ErrTokenInvalid       = 20003
	ErrRefreshTokenReuse  = 20004
	ErrInvalidCredentials = 20005

	// Forbidden / CORS (3xxxx).
	ErrForbidden        = 30001
	ErrOriginNotAllowed = 30002
	ErrUserDisabled     = 30003

	// Not found (4xxxx).
	ErrNotFound               = 40001
	ErrFeedbackNotFound       = 40002
	ErrUserNotFound           = 40003
	ErrRoleNotFound           = 40004
	ErrPermissionNotFound     = 40005
	ErrTaskNotFound           = 40006
	ErrDeviceNotFound         = 40007
	ErrFileNotFound           = 40008
	ErrUploadSessionNotFound  = 40009
	ErrDeviceGroupNotFound    = 40010
	ErrLicenseNotFound        = 40011
	ErrAITimeScheduleNotFound = 40012

	// System management errors (10xxx).
	ErrCleanupRunning        = 10030
	ErrCleanupDisabled       = 10031
	ErrTimeSyncFailed        = 10032
	ErrInvalidTimezone       = 10033
	ErrTimezoneFileNotFound  = 10034
	ErrSetTimeFailed         = 10035
	ErrTimeOutOfRange        = 10036
	ErrNetworkConfigFailed   = 10037
	ErrNetworkRollbackFailed = 10038
	ErrNetworkConfirmFailed  = 10039
	ErrWebhookPushFailed     = 10040
	ErrNotImplemented        = 10041
	ErrTooManyRequests       = 10042

	// License 算法授权错误 (10xxx)
	ErrLicenseInvalid        = 10050 // 授权文件无效或签名验证失败
	ErrLicenseDeviceMismatch = 10051 // 授权文件绑定的设备指纹与当前设备不匹配
	ErrLicenseNotAuthorized  = 10052 // 算法未在授权范围内
	ErrLicenseExpired        = 10053 // 授权已过期
	ErrLicenseNotYetValid    = 10054 // 授权尚未生效
	ErrLicenseStreamLimit    = 10055 // 授权并发路数不足
	ErrLicenseNotConfigured  = 10056 // 授权公钥未配置

	// 连接测试结果 (1008x).
	ErrEngineNotReady     = 10080 // 推理引擎未就绪
	ErrConnectionTimeout  = 10081 // 连接超时
	ErrEngineResponseBad  = 10082 // 引擎响应异常
	ErrConnectionFailed   = 10083 // 连接失败
	ConnectionTestSuccess = 10084 // 连接测试成功，流可达

	// Server errors (5xxxx).
	ErrInternal            = 50001
	ErrStreamStateNotFound = 50002
	ErrZLMNotConfigured    = 50003
	ErrFingerprintExtract  = 50004
)

const defaultLanguage = "en"

// AppError represents a business-level error with a code and message.
type AppError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`

	// localizedMessage 标记 Message 是否已经按当前请求语言翻译，可直接返回给前端。
	localizedMessage bool
}

// Error implements the error interface.
func (e *AppError) Error() string {
	return fmt.Sprintf("code=%d, message=%s", e.Code, e.Message)
}

// New creates a new AppError with the given code and message.
// If the message is empty, the default message for the code is used.
func New(code int, msg string) *AppError {
	if msg == "" {
		msg = DefaultMessage(code, defaultLanguage)
	}
	return &AppError{Code: code, Message: msg}
}

// IsLocalizedMessage reports whether the message was already translated for the current request.
func (e *AppError) IsLocalizedMessage() bool {
	return e != nil && e.localizedMessage
}

// NewLocalized creates an AppError whose explicit message is safe to return to the frontend.
func NewLocalized(code int, msg string) *AppError {
	if msg == "" {
		return New(code, msg)
	}
	return &AppError{Code: code, Message: msg, localizedMessage: true}
}

// Newf creates a new AppError with a formatted explicit message.
func Newf(code int, format string, args ...interface{}) *AppError {
	return &AppError{Code: code, Message: fmt.Sprintf(format, args...)}
}

// DefaultMessage returns the default message for a business error code in the specified language.
func DefaultMessage(code int, lang string) string {
	msg := i18n.Translate(lang, code)
	if strings.HasPrefix(msg, "unknown error (code=") {
		return "Unknown error"
	}
	return msg
}
