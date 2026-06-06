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

	// Person management errors (1006x).
	ErrPersonImageRequired   = 10060 // 请上传人脸图片
	ErrPersonImageDuplicate  = 10061 // 图片已存在
	ErrPersonCodeDuplicate   = 10062 // 人员编号已存在
	ErrPersonStatusNoRetry   = 10063 // 当前状态无需重提
	ErrArchiveRequired       = 10064 // 请上传压缩包
	ErrArchiveUnsupported    = 10065 // 不支持的压缩包格式

	// Auth errors (2xxxx).
	ErrUnauthorized      = 20001
	ErrTokenExpired      = 20002
	ErrTokenInvalid      = 20003
	ErrRefreshTokenReuse = 20004

	// Forbidden / CORS (3xxxx).
	ErrForbidden        = 30001
	ErrOriginNotAllowed = 30002

	// Not found (4xxxx).
	ErrNotFound         = 40001
	ErrFeedbackNotFound = 40002

	// System management errors (10xxx).
	ErrCleanupRunning         = 10030
	ErrCleanupDisabled        = 10031
	ErrTimeSyncFailed         = 10032
	ErrInvalidTimezone        = 10033
	ErrTimezoneFileNotFound   = 10034
	ErrSetTimeFailed          = 10035
	ErrTimeOutOfRange         = 10036
	ErrNetworkConfigFailed    = 10037
	ErrNetworkRollbackFailed  = 10038
	ErrNetworkConfirmFailed   = 10039
	ErrWebhookPushFailed      = 10040
	ErrNotImplemented         = 10041
	ErrTooManyRequests        = 10042

	// License 算法授权错误 (10xxx)
	ErrLicenseInvalid          = 10050 // 授权文件无效或签名验证失败
	ErrLicenseDeviceMismatch   = 10051 // 授权文件绑定的设备指纹与当前设备不匹配
	ErrLicenseNotAuthorized    = 10052 // 算法未在授权范围内
	ErrLicenseExpired          = 10053 // 授权已过期
	ErrLicenseNotYetValid      = 10054 // 授权尚未生效
	ErrLicenseStreamLimit      = 10055 // 授权并发路数不足
	ErrLicenseNotConfigured    = 10056 // 授权公钥未配置

	// Server errors (5xxxx).
	ErrInternal = 50001
)

const defaultLanguage = "en"

// AppError represents a business-level error with a code and message.
type AppError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`

	// defaultMessage 标记 Message 是否由错误码默认文案生成，避免响应层通过字符串比较判断。
	defaultMessage bool
}

// Error implements the error interface.
func (e *AppError) Error() string {
	return fmt.Sprintf("code=%d, message=%s", e.Code, e.Message)
}

// New creates a new AppError with the given code and message.
// If the message is empty, the default message for the code is used.
func New(code int, msg string) *AppError {
	isDefault := msg == ""
	if isDefault {
		msg = DefaultMessage(code, defaultLanguage)
	}
	return &AppError{Code: code, Message: msg, defaultMessage: isDefault}
}

// IsDefaultMessage reports whether the message was generated from the default code mapping.
func (e *AppError) IsDefaultMessage() bool {
	return e != nil && e.defaultMessage
}

// Newf creates a new AppError with a formatted explicit message.
func Newf(code int, format string, args ...interface{}) *AppError {
	return &AppError{Code: code, Message: fmt.Sprintf(format, args...), defaultMessage: false}
}

// DefaultMessage returns the default message for a business error code in the specified language.
func DefaultMessage(code int, lang string) string {
	msg := i18n.Translate(lang, code)
	if strings.HasPrefix(msg, "unknown error (code=") {
		return "Unknown error"
	}
	return msg
}
