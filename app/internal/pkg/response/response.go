// Package response provides unified JSON response formatting for the
// niko-admin application.
package response

import (
	"net/http"

	"github.com/gin-gonic/gin"

	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
	"github.com/niko-admin/niko-admin/internal/pkg/i18n"
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

// OK sends a success response with code=0.
func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    apperrors.Success,
		Message: i18n.Translate(c.GetString("lang"), apperrors.Success),
		Data:    data,
	})
}

// Err sends an error response based on the AppError code.
func Err(c *gin.Context, err error) {
	if appErr, ok := err.(*apperrors.AppError); ok {
		httpStatus := codeToHTTPStatus(appErr.Code)
		c.JSON(httpStatus, Response{
			Code:    appErr.Code,
			Message: i18n.Translate(c.GetString("lang"), appErr.Code),
		})
		return
	}
	c.JSON(http.StatusInternalServerError, Response{
		Code:    apperrors.ErrInternal,
		Message: i18n.Translate(c.GetString("lang"), apperrors.ErrInternal),
	})
}

// Page sends a paginated success response.
func Page(c *gin.Context, list interface{}, total int64, page, pageSize int) {
	c.JSON(http.StatusOK, Response{
		Code:    apperrors.Success,
		Message: i18n.Translate(c.GetString("lang"), apperrors.Success),
		Data: PageData{
			List:     list,
			Total:    total,
			Page:     page,
			PageSize: pageSize,
		},
	})
}

// codeToHTTPStatus maps business error codes to HTTP status codes.
func codeToHTTPStatus(code int) int {
	switch {
	case code >= 10000 && code < 20000:
		return http.StatusBadRequest // 1xxxx -> 400
	case code >= 20000 && code < 30000:
		return http.StatusUnauthorized // 2xxxx -> 401
	case code >= 30000 && code < 40000:
		return http.StatusForbidden // 3xxxx -> 403
	case code >= 40000 && code < 50000:
		return http.StatusNotFound // 4xxxx -> 404
	case code >= 50000 && code < 60000:
		return http.StatusInternalServerError // 5xxxx -> 500
	default:
		return http.StatusOK
	}
}
