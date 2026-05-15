package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
)

// permission represents a single permission record from the database.
type permission struct {
	Path   string `json:"path"`
	Method string `json:"method"`
}

const (
	permCachePrefix = "perm:"
	permCacheTTL    = 5 * time.Minute
)

// RBAC returns a Gin middleware factory that checks whether the current user
// has permission for the requested path and method. It reads user_id from
// the gin.Context (set by the Auth middleware). The permission is resolved
// by calling CheckPermission internally.
func RBAC(rdb *redis.Client, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get(ContextKeyUserID)
		if !exists {
			c.Error(apperrors.New(apperrors.ErrUnauthorized, ""))
			c.Abort()
			return
		}

		uid, ok := userID.(string)
		if !ok || uid == "" {
			c.Error(apperrors.New(apperrors.ErrUnauthorized, ""))
			c.Abort()
			return
		}

		path := c.Request.URL.Path
		method := c.Request.Method

		allowed, err := CheckPermission(rdb, db, uid, path, method)
		if err != nil {
			zap.L().Error("rbac check failed",
				zap.String("user_id", uid),
				zap.String("path", path),
				zap.String("method", method),
				zap.Error(err),
			)
			c.Error(apperrors.New(apperrors.ErrInternal, ""))
			c.Abort()
			return
		}

		if !allowed {
			zap.L().Warn("permission denied",
				zap.String("user_id", uid),
				zap.String("path", path),
				zap.String("method", method),
			)
			c.Error(apperrors.New(apperrors.ErrForbidden, ""))
			c.Abort()
			return
		}

		c.Next()
	}
}

// CheckPermission verifies whether the given user has a permission matching
// the specified path and method. It checks Redis cache first, falling back
// to the database on cache miss.
func CheckPermission(rdb *redis.Client, db *gorm.DB, userID, path, method string) (bool, error) {
	ctx := context.Background()
	cacheKey := fmt.Sprintf("%s%s", permCachePrefix, userID)

	// Try cache first
	cached, err := rdb.Get(ctx, cacheKey).Bytes()
	if err == nil && len(cached) > 0 {
		var perms []permission
		if err := json.Unmarshal(cached, &perms); err == nil {
			return matchPermission(perms, path, method), nil
		}
	}

	// Cache miss — query database
	// Query user's permissions through the role_permissions / user_roles join.
	// Schema assumption:
	//   - users (id)
	//   - user_roles (user_id, role_id)
	//   - role_permissions (role_id, permission_id)
	//   - permissions (id, path, method)
	var perms []permission
	query := `
		SELECT DISTINCT p.path, p.method
		FROM permissions p
		INNER JOIN role_permissions rp ON rp.permission_id = p.id
		INNER JOIN user_roles ur ON ur.role_id = rp.role_id
		WHERE ur.user_id = ?
	`
	if err := db.Raw(query, userID).Scan(&perms).Error; err != nil {
		return false, fmt.Errorf("failed to query permissions: %w", err)
	}

	// Update cache
	if data, err := json.Marshal(perms); err == nil {
		if err := rdb.Set(ctx, cacheKey, data, permCacheTTL).Err(); err != nil {
			zap.L().Warn("failed to cache permissions",
				zap.String("user_id", userID),
				zap.Error(err),
			)
		}
	}

	return matchPermission(perms, path, method), nil
}

// matchPermission checks if any permission in the list matches the given
// path and method. An empty path in the permission record acts as a wildcard
// for the method (used for "access all" permissions).
func matchPermission(perms []permission, path, method string) bool {
	for _, p := range perms {
		if p.Path == "*" {
			return true
		}
		if p.Path == path && (p.Method == "*" || p.Method == method) {
			return true
		}
	}
	return false
}
