package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
	jwtutil "github.com/niko-admin/niko-admin/internal/pkg/jwt"
)

const (
	// ContextKeyUserID is the gin context key for the authenticated user's ID.
	ContextKeyUserID = "user_id"
	// ContextKeyRoleIDs is the gin context key for the user's role IDs.
	ContextKeyRoleIDs = "role_ids"
)

// Auth returns a Gin middleware that extracts and validates a Bearer token
// from the Authorization header. On success it sets "user_id" and "role_ids"
// in the gin.Context. On failure it calls c.Abort() and c.Error().
func Auth(jwtManager *jwtutil.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Error(apperrors.New(apperrors.ErrUnauthorized, "缺少认证令牌"))
			c.Abort()
			return
		}

		// Extract Bearer token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.Error(apperrors.New(apperrors.ErrUnauthorized, "认证格式错误"))
			c.Abort()
			return
		}

		tokenString := parts[1]

		// Validate token
		claims, err := jwtManager.ValidateAccessToken(tokenString)
		if err != nil {
			zap.L().Debug("token validation failed", zap.Error(err))
			c.Error(apperrors.New(apperrors.ErrTokenInvalid, "令牌无效或已过期"))
			c.Abort()
			return
		}

		// Set user info in context
		c.Set(ContextKeyUserID, claims.UserID)
		c.Set(ContextKeyRoleIDs, claims.RoleIDs)

		c.Next()
	}
}
