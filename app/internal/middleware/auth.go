package middleware

import (
	"context"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/niko-admin/niko-admin/internal/model"
	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
	jwtutil "github.com/niko-admin/niko-admin/internal/pkg/jwt"
)

const (
	// ContextKeyUserID is used only for gin.Context.Set/Get in HTTP handlers/middleware.
	// Do not use it with context.WithValue; for request context use model.ContextKeyUserID.
	ContextKeyUserID = "user_id"
	// ContextKeyRoleIDs is the gin context key for the user's role IDs.
	ContextKeyRoleIDs = "role_ids"
	// ContextKeyIsRoot is the gin context key for whether the user is a root/superadmin.
	ContextKeyIsRoot = "is_root"
)

// Auth returns a Gin middleware that extracts and validates a Bearer token
// from the Authorization header. On success it sets "user_id" and "role_ids"
// in the gin.Context. On failure it calls c.Abort() and c.Error().
func Auth(jwtManager *jwtutil.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			abortWithError(c, apperrors.New(apperrors.ErrUnauthorized, "缺少认证令牌"))
			return
		}

		// Extract Bearer token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			abortWithError(c, apperrors.New(apperrors.ErrUnauthorized, "认证格式错误"))
			return
		}

		tokenString := parts[1]

		// Validate token
		claims, err := jwtManager.ValidateAccessToken(tokenString)
		if err != nil {
			zap.L().Debug("token validation failed", zap.Error(err))
			abortWithError(c, apperrors.New(apperrors.ErrTokenInvalid, "令牌无效或已过期"))
			return
		}

		// Set user info in gin context
		c.Set(ContextKeyUserID, claims.UserID)
		c.Set(ContextKeyRoleIDs, claims.RoleIDs)
		c.Set(ContextKeyIsRoot, claims.IsRoot)

		// Set user info in request context (for GORM hooks)
		ctx := c.Request.Context()
		ctx = context.WithValue(ctx, model.ContextKeyUserID, claims.UserID)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

// abortWithError 挂载业务错误并终止当前请求链路。
func abortWithError(c *gin.Context, err error) {
	if attachErr := c.Error(err); attachErr != nil {
		zap.L().Warn("attach gin error failed", zap.Error(attachErr))
	}
	c.Abort()
}
