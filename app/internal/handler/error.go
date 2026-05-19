package handler

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/niko-admin/niko-admin/internal/middleware"
	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
	validatorx "github.com/niko-admin/niko-admin/internal/pkg/validator"
)

// attachError 将错误挂载到 Gin Context，供全局错误中间件统一响应。
func attachError(c *gin.Context, err error) {
	if attachErr := c.Error(err); attachErr != nil {
		zap.L().Warn("attach gin error failed", zap.Error(attachErr))
	}
}

// badRequestError wraps request binding/validation errors into a localized AppError.
func badRequestError(c *gin.Context, err error) error {
	// lang key is injected by i18n middleware; if missing, validator layer defaults to English.
	lang, _ := c.Get(middleware.ContextKeyLang)
	langStr, _ := lang.(string)
	return apperrors.New(apperrors.ErrBadRequest, validatorx.TranslateValidationError(err, langStr))
}

// getUserID extracts the authenticated user ID from the Gin context.
// Returns the user ID and true on success, or empty string and false on failure
// (in which case an unauthorized error is attached to the context).
func getUserID(c *gin.Context) (string, bool) {
	userID, exists := c.Get(middleware.ContextKeyUserID)
	if !exists {
		attachError(c, apperrors.New(apperrors.ErrUnauthorized, ""))
		return "", false
	}
	uid, ok := userID.(string)
	if !ok || uid == "" {
		attachError(c, apperrors.New(apperrors.ErrUnauthorized, ""))
		return "", false
	}
	return uid, true
}
