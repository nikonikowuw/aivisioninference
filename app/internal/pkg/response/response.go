// Package response provides unified JSON response formatting for the
// niko-admin application.
package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
)

// Response is the standard API response wrapper.
type Response struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// PageData is the paginated response data structure.
type PageData struct {
	List     interface{} `json:"list"`
	Total    int64       `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
}

const successMessage = "success"

// OK sends a success response with code="OK".
func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    apperrors.Success,
		Message: successMessage,
		Data:    data,
	})
}

// Err 根据 AppError 错误码返回统一错误响应。
func Err(c *gin.Context, err error) {
	appErr, ok := err.(*apperrors.AppError)
	if !ok {
		zap.L().Error("unexpected non-app error response", zap.Error(err))
		appErr = apperrors.New(apperrors.ErrInternal, "")
	}

	message := appErr.Message
	if !appErr.IsLocalizedMessage() {
		message = apperrors.DefaultMessage(appErr.Code, contextLanguage(c))
	}

	c.JSON(codeToHTTPStatus(appErr.Code), Response{
		Code:    appErr.Code,
		Message: message,
	})
}

// Page sends a paginated success response.
func Page(c *gin.Context, list interface{}, total int64, page, pageSize int) {
	c.JSON(http.StatusOK, Response{
		Code:    apperrors.Success,
		Message: successMessage,
		Data: PageData{
			List:     list,
			Total:    total,
			Page:     page,
			PageSize: pageSize,
		},
	})
}

// contextLanguage extracts the language from the context.
func contextLanguage(c *gin.Context) string {
	if lang, ok := c.Get("lang"); ok {
		if langStr, ok := lang.(string); ok && langStr != "" {
			return langStr
		}
	}
	return "en"
}

// codeToHTTPStatus maps business error codes to HTTP status codes.
func codeToHTTPStatus(code string) int {
	if s, ok := codeToHTTPStatusMap[code]; ok {
		return s
	}
	return http.StatusOK
}

// codeToHTTPStatusMap is the explicit mapping from error code strings to HTTP status codes.
var codeToHTTPStatusMap = map[string]int{
	// REQ — 400 Bad Request
	"REQ_BAD_REQUEST":        http.StatusBadRequest,
	"REQ_CANNOT_DISABLE_SELF": http.StatusBadRequest,
	"REQ_START_TIME_FORMAT":  http.StatusBadRequest,
	"REQ_END_TIME_FORMAT":    http.StatusBadRequest,
	"REQ_TIME_RANGE_ORDER":   http.StatusBadRequest,
	"REQ_FILE_TOO_LARGE":     http.StatusBadRequest,
	"REQ_FILE_INVALID_TYPE":  http.StatusBadRequest,
	"REQ_NOT_IMPLEMENTED":    http.StatusNotImplemented,
	"REQ_TOO_MANY_REQUESTS":  http.StatusTooManyRequests,

	// AUTH — 401 Unauthorized
	"AUTH_UNAUTHORIZED":        http.StatusUnauthorized,
	"AUTH_TOKEN_EXPIRED":       http.StatusUnauthorized,
	"AUTH_TOKEN_INVALID":       http.StatusUnauthorized,
	"AUTH_TOKEN_REUSED":        http.StatusUnauthorized,
	"AUTH_INVALID_CREDENTIALS": http.StatusUnauthorized,

	// FORBIDDEN — 403
	"FORBIDDEN":          http.StatusForbidden,
	"FORBIDDEN_ORIGIN":   http.StatusForbidden,
	"FORBIDDEN_USER_DISABLED": http.StatusForbidden,

	// NOT_FOUND — 404
	"NOT_FOUND":               http.StatusNotFound,
	"NOT_FOUND_FEEDBACK":      http.StatusNotFound,
	"NOT_FOUND_USER":          http.StatusNotFound,
	"NOT_FOUND_ROLE":          http.StatusNotFound,
	"NOT_FOUND_PERMISSION":    http.StatusNotFound,
	"NOT_FOUND_TASK":          http.StatusNotFound,
	"NOT_FOUND_DEVICE":        http.StatusNotFound,
	"NOT_FOUND_FILE":          http.StatusNotFound,
	"NOT_FOUND_UPLOAD_SESSION": http.StatusNotFound,
	"NOT_FOUND_DEVICE_GROUP":  http.StatusNotFound,
	"NOT_FOUND_LICENSE":       http.StatusNotFound,
	"NOT_FOUND_TIME_SCHEDULE": http.StatusNotFound,

	// DEV — 409 Conflict (resource conflict) or 400
	"DEV_RESOURCE_CONFLICT": http.StatusConflict,

	// USER, ROLE, CSV, PERSON, PERM, FILE, SYS, LICENSE, ENGINE, DEV, NODE, INTERNAL — 400 Bad Request
	"USER_EMAIL_TAKEN":             http.StatusBadRequest,
	"USER_OLD_PASSWORD_WRONG":      http.StatusBadRequest,
	"USER_HIERARCHY_LEVEL":         http.StatusBadRequest,
	"USER_MAIL_NOT_ENABLED":        http.StatusBadRequest,
	"USER_SMTP_TEST_FAILED":        http.StatusBadRequest,
	"USER_IMAP_TEST_FAILED":        http.StatusBadRequest,
	"USER_TOKEN_INVALID":           http.StatusBadRequest,
	"USER_USERNAME_TAKEN":          http.StatusBadRequest,
	"USER_CANNOT_DELETE_SELF":      http.StatusBadRequest,
	"USER_CANNOT_RESET_SELF":       http.StatusBadRequest,
	"ROLE_HIERARCHY_LEVEL":         http.StatusBadRequest,
	"ROLE_NAME_TAKEN":              http.StatusBadRequest,
	"ROLE_LEVEL_INVALID":           http.StatusBadRequest,
	"CSV_INVALID_CONTENT":          http.StatusBadRequest,
	"CSV_ROW_LIMIT_EXCEEDED":       http.StatusBadRequest,
	"CSV_COLUMN_REQUIRED":          http.StatusBadRequest,
	"CSV_STATUS_INVALID":           http.StatusBadRequest,
	"CSV_HEADER_INVALID":           http.StatusBadRequest,
	"CSV_DUPLICATE_USERNAME":       http.StatusBadRequest,
	"CSV_WEAK_PASSWORD":            http.StatusBadRequest,
	"CSV_INVALID_EMAIL":            http.StatusBadRequest,
	"PERSON_IMAGE_REQUIRED":        http.StatusBadRequest,
	"PERSON_IMAGE_DUPLICATE":       http.StatusBadRequest,
	"PERSON_CODE_DUPLICATE":        http.StatusBadRequest,
	"PERSON_STATUS_NO_RETRY":       http.StatusBadRequest,
	"PERSON_ARCHIVE_REQUIRED":      http.StatusBadRequest,
	"PERSON_ARCHIVE_UNSUPPORTED":   http.StatusBadRequest,
	"PERSON_FACE_EXTRACT_FAILED":   http.StatusBadRequest,
	"PERSON_FACE_SEARCH_NO_RESULT": http.StatusBadRequest,
	"PERM_CODE_TAKEN":              http.StatusBadRequest,
	"PERM_ASSIGNED":                http.StatusBadRequest,
	"PERM_ROLE_ASSIGNED":           http.StatusBadRequest,
	"FILE_CONTENT_INCOMPLETE":      http.StatusBadRequest,
	"FILE_CHUNK_COUNT_MISMATCH":    http.StatusBadRequest,
	"FILE_INVALID_CHUNK_INDEX":     http.StatusBadRequest,
	"FILE_CHUNK_INCOMPLETE":        http.StatusBadRequest,
	"FILE_UPLOAD_CANCELED":         http.StatusBadRequest,
	"FILE_SIZE_MISMATCH":           http.StatusBadRequest,
	"FILE_CHECKSUM_MISMATCH":       http.StatusBadRequest,
	"SYS_CLEANUP_RUNNING":          http.StatusBadRequest,
	"SYS_CLEANUP_DISABLED":         http.StatusBadRequest,
	"SYS_TIME_SYNC_FAILED":         http.StatusBadRequest,
	"SYS_INVALID_TIMEZONE":         http.StatusBadRequest,
	"SYS_TIMEZONE_FILE_NOT_FOUND":  http.StatusBadRequest,
	"SYS_SET_TIME_FAILED":          http.StatusBadRequest,
	"SYS_TIME_OUT_OF_RANGE":        http.StatusBadRequest,
	"SYS_NETWORK_CONFIG_FAILED":    http.StatusBadRequest,
	"SYS_NETWORK_ROLLBACK_FAILED":  http.StatusBadRequest,
	"SYS_NETWORK_CONFIRM_FAILED":   http.StatusBadRequest,
	"SYS_WEBHOOK_PUSH_FAILED":      http.StatusBadRequest,
	"LICENSE_INVALID":              http.StatusBadRequest,
	"LICENSE_DEVICE_MISMATCH":      http.StatusBadRequest,
	"LICENSE_NOT_AUTHORIZED":       http.StatusBadRequest,
	"LICENSE_EXPIRED":              http.StatusBadRequest,
	"LICENSE_NOT_YET_VALID":        http.StatusBadRequest,
	"LICENSE_STREAM_LIMIT":         http.StatusBadRequest,
	"LICENSE_NOT_CONFIGURED":       http.StatusBadRequest,
	"ENGINE_NOT_READY":             http.StatusBadRequest,
	"ENGINE_CONNECTION_TIMEOUT":    http.StatusGatewayTimeout,
	"ENGINE_RESPONSE_BAD":          http.StatusBadGateway,
	"ENGINE_CONNECTION_FAILED":     http.StatusBadGateway,
	"ENGINE_CONNECTION_TEST_OK":    http.StatusOK,
	"DEV_NAME_TAKEN":               http.StatusBadRequest,
	"DEV_RTSP_URL_REQUIRED":        http.StatusBadRequest,
	"DEV_GB28181_CODE_REQUIRED":    http.StatusBadRequest,
	"DEV_GROUP_NOT_EMPTY":          http.StatusBadRequest,
	"DEV_GB28181_CODE_INVALID":     http.StatusBadRequest,
	"DEV_GB28181_OFFLINE":          http.StatusBadRequest,
	"DEV_INVALID_PLAYBACK_ACTION":  http.StatusBadRequest,
	"DEV_ALARM_DEVICE_REQUIRED":    http.StatusBadRequest,
	"DEV_TIME_WINDOW_FORMAT":       http.StatusBadRequest,
	"DEV_TASK_STATUS_NOT_CANCELABLE": http.StatusBadRequest,
	"DEV_EXTERNAL_KEY_TAKEN":       http.StatusBadRequest,
	"DEV_DISABLED":                 http.StatusBadRequest,
	"DEV_OFFLINE":                  http.StatusBadRequest,
	"DEV_TYPE_INVALID":             http.StatusBadRequest,
	"DEV_VERSION_INCOMPATIBLE":     http.StatusBadRequest,
	"NODE_NAME_TAKEN":              http.StatusBadRequest,
	"NODE_NOT_FOUND":               http.StatusNotFound,
	"NODE_OFFLINE":                 http.StatusBadRequest,
	"NODE_DISABLED":                http.StatusBadRequest,
	"NODE_FULL":                    http.StatusBadRequest,
	"NODE_ALGORITHM_UNAVAILABLE":   http.StatusBadRequest,

	// INTERNAL — 500
	"INTERNAL_ERROR":                   http.StatusInternalServerError,
	"INTERNAL_STREAM_STATE_NOT_FOUND":  http.StatusInternalServerError,
	"INTERNAL_ZLM_NOT_CONFIGURED":      http.StatusInternalServerError,
	"INTERNAL_FINGERPRINT_EXTRACT_FAIL": http.StatusInternalServerError,
}
