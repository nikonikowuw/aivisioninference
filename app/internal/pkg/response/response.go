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
	Code    int         `json:"code"`
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

// OK sends a success response with code=0.
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
func codeToHTTPStatus(code int) int {
	if code == apperrors.ErrResourceConflict {
		return http.StatusConflict
	}

	switch code / 10000 {
	case 1:
		if code == apperrors.ErrTooManyRequests {
			return http.StatusTooManyRequests
		}
		return http.StatusBadRequest // 1xxxx -> 400
	case 2:
		return http.StatusUnauthorized // 2xxxx -> 401
	case 3:
		return http.StatusForbidden // 3xxxx -> 403
	case 4:
		return http.StatusNotFound // 4xxxx -> 404
	case 5:
		return http.StatusInternalServerError // 5xxxx -> 500
	default:
		return http.StatusOK
	}
}
