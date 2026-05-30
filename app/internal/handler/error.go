// Package handler 提供 HTTP 请求处理层（Controller），负责参数绑定、校验和响应返回。
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

// badRequestError 将请求绑定/校验错误封装为本地化翻译的 AppError。
func badRequestError(c *gin.Context, err error) error {
	lang, _ := c.Get(middleware.ContextKeyLang)
	langStr, _ := lang.(string)
	return apperrors.New(apperrors.ErrBadRequest, validatorx.TranslateValidationError(err, langStr))
}

// currentLang 从 Gin 上下文中提取中间件注入的当前语言设置。
func currentLang(c *gin.Context) string {
	lang, _ := c.Get(middleware.ContextKeyLang)
	langStr, _ := lang.(string)
	return langStr
}

// currentUserContext 从 Gin 上下文中提取当前登录用户 ID 和是否为超级管理员。
func currentUserContext(c *gin.Context) (string, bool) {
	currentUserID, _ := c.Get(middleware.ContextKeyUserID)
	uid, _ := currentUserID.(string)
	isRootVal, _ := c.Get(middleware.ContextKeyIsRoot)
	isRoot, _ := isRootVal.(bool)
	return uid, isRoot
}

// getUserID 从 Gin 上下文中提取已认证的用户 ID。
// 成功时返回用户 ID 和 true；失败时返回空字符串和 false，并自动在上下文中挂载未授权错误。
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
