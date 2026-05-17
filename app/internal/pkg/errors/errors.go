// Package errors provides unified business error codes and error types
// for the niko-admin application.
package errors

import "fmt"

// Standard business error codes.
const (
	Success = 0

	// Client errors (1xxxx).
	ErrBadRequest         = 10001
	ErrCannotDisableSelf  = 10002
	ErrHierarchyLevelUser = 10003
	ErrHierarchyLevelRole = 10004
	ErrEmailTaken         = 10005
	ErrOldPasswordWrong   = 10006
	ErrStartTimeFormat     = 10008
	ErrEndTimeFormat       = 10009
	ErrTimeRangeOrder      = 10010
	ErrFileTooLarge        = 10011
	ErrFileInvalidType     = 10012

	// Auth errors (2xxxx).
	ErrUnauthorized      = 20001
	ErrTokenExpired      = 20002
	ErrTokenInvalid      = 20003
	ErrRefreshTokenReuse = 20403

	// Forbidden (3xxxx).
	ErrForbidden = 30001

	// Not found (4xxxx).
	ErrNotFound = 40001

	// Server errors (5xxxx).
	ErrInternal = 50001
)

// Standard error messages keyed by code.
var messages = map[int]string{
	Success:               "success",
	ErrBadRequest:         "请求参数错误",
	ErrCannotDisableSelf:  "不能禁用自己",
	ErrHierarchyLevelUser: "没有权限操作同级或更高级别的用户",
	ErrHierarchyLevelRole: "没有权限操作同级或更高级别的角色，也不能设置高于自己权限的角色等级",
	ErrEmailTaken:         "邮箱已被使用",
	ErrOldPasswordWrong:   "旧密码错误",
	ErrStartTimeFormat:    "开始时间格式错误",
	ErrEndTimeFormat:      "结束时间格式错误",
	ErrTimeRangeOrder:     "开始时间不能晚于结束时间",
	ErrFileTooLarge:       "文件大小超过限制",
	ErrFileInvalidType:    "不支持的文件类型",
	ErrUnauthorized:       "未登录",
	ErrTokenExpired:       "Token已过期",
	ErrTokenInvalid:       "Token无效",
	ErrRefreshTokenReuse:  "Token已被复用",
	ErrForbidden:          "无权限",
	ErrNotFound:           "资源不存在",
	ErrInternal:           "服务器内部错误",
}

// AppError represents a business-level error with a code and message.
type AppError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Error implements the error interface.
func (e *AppError) Error() string {
	return fmt.Sprintf("code=%d, message=%s", e.Code, e.Message)
}

// New creates a new AppError with the given code and message.
// If the message is empty, the default message for the code is used.
func New(code int, msg string) *AppError {
	if msg == "" {
		if m, ok := messages[code]; ok {
			msg = m
		} else {
			msg = "未知错误"
		}
	}
	return &AppError{Code: code, Message: msg}
}

// Newf creates a new AppError with a formatted message.
func Newf(code int, format string, args ...interface{}) *AppError {
	return New(code, fmt.Sprintf(format, args...))
}
