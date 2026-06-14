package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
	"github.com/niko-admin/niko-admin/internal/pkg/jwt"
)

// EdgeNodeMiddleware handles authentication for edge inference engines.
type EdgeNodeMiddleware struct {
	jwtManager *jwt.Manager
}

// NewEdgeNodeMiddleware creates a new EdgeNodeMiddleware.
func NewEdgeNodeMiddleware(jwtManager *jwt.Manager) *EdgeNodeMiddleware {
	return &EdgeNodeMiddleware{jwtManager: jwtManager}
}

func (m *EdgeNodeMiddleware) AuthNode() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			abortWithError(c, apperrors.New(apperrors.ErrUnauthorized, "缺少认证令牌"))
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			abortWithError(c, apperrors.New(apperrors.ErrUnauthorized, "认证格式错误"))
			return
		}

		claims, err := m.jwtManager.ParseNodeToken(parts[1])
		if err != nil {
			zap.L().Debug("node token validation failed", zap.Error(err))
			abortWithError(c, apperrors.New(apperrors.ErrUnauthorized, "令牌无效或已过期"))
			return
		}

		pathNodeID := c.Param("id")
		if pathNodeID != "" && pathNodeID != claims.NodeID {
			zap.L().Warn("node id mismatch", 
				zap.String("path_node_id", pathNodeID), 
				zap.String("token_node_id", claims.NodeID))
			abortWithError(c, apperrors.New(apperrors.ErrForbidden, "节点 ID 不匹配"))
			return
		}

		c.Set("node_id", claims.NodeID)
		c.Next()
	}
}
