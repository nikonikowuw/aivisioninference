package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/niko-admin/niko-admin/internal/model"
	cachepkg "github.com/niko-admin/niko-admin/internal/pkg/cache"
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
func RBAC(cache cachepkg.Cache, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get(ContextKeyUserID)
		if !exists {
			abortWithError(c, apperrors.New(apperrors.ErrUnauthorized, ""))
			return
		}

		uid, ok := userID.(string)
		if !ok || uid == "" {
			abortWithError(c, apperrors.New(apperrors.ErrUnauthorized, ""))
			return
		}

		// Skip RBAC for root user — is_root is set by Auth middleware from JWT claims
		if isRoot, _ := c.Get(ContextKeyIsRoot); isRoot != nil {
			if root, ok := isRoot.(bool); ok && root {
				c.Next()
				return
			}
		}

		reqPath := path.Clean(c.Request.URL.Path)
		method := c.Request.Method

		allowed, err := CheckPermission(c.Request.Context(), cache, db, uid, reqPath, method)
		if err != nil {
			zap.L().Error("rbac check failed",
				zap.String("user_id", uid),
				zap.String("req_path", reqPath),
				zap.String("method", method),
				zap.Error(err),
			)
			abortWithError(c, apperrors.New(apperrors.ErrInternal, ""))
			return
		}

		if !allowed {
			zap.L().Warn("permission denied",
				zap.String("user_id", uid),
				zap.String("req_path", reqPath),
				zap.String("method", method),
			)
			abortWithError(c, apperrors.New(apperrors.ErrForbidden, ""))
			return
		}

		c.Next()
	}
}

// CheckPermission verifies whether the given user has a permission matching
// the specified path and method. It checks Redis cache first, falling back
// to the database on cache miss.
func CheckPermission(ctx context.Context, cache cachepkg.Cache, db *gorm.DB, userID, path, method string) (bool, error) {
	cacheKey := fmt.Sprintf("%s%s", permCachePrefix, userID)

	// Try cache first
	if cache != nil {
		cached, err := cache.Get(ctx, cacheKey)
		if err == nil && len(cached) > 0 {
			var perms []permission
			if err := json.Unmarshal(cached, &perms); err == nil {
				return matchPermission(perms, path, method), nil
			}
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
		AND p.type != ?
	`
	if err := db.WithContext(ctx).Raw(query, userID, model.PermTypeMenu).Scan(&perms).Error; err != nil {
		return false, fmt.Errorf("failed to query permissions: %w", err)
	}

	// Update cache
	if cache != nil {
		if data, err := json.Marshal(perms); err == nil {
			if err := cache.Set(ctx, cacheKey, data, permCacheTTL); err != nil {
				zap.L().Warn("failed to cache permissions",
					zap.String("user_id", userID),
					zap.Error(err),
				)
			}
		}
	}

	return matchPermission(perms, path, method), nil
}

// matchPermission checks if any permission in the list matches the given
// path and method.
//
// Matching rules:
//   - Path "*" matches all paths (super-admin).
//   - Method "*" matches all HTTP methods.
//   - A path segment "*" matches exactly one path segment
//     (e.g. "/api/v1/roles/*" matches "/api/v1/roles/42"
//     but NOT "/api/v1/roles/42/permissions").
//   - A trailing "/**" matches all subpaths
//     (e.g. "/api/v1/files/upload/**" matches
//     "/api/v1/files/upload/init" and "/api/v1/files/upload/123/chunk").
//   - Otherwise, the path must match exactly.
func matchPermission(perms []permission, reqPath, method string) bool {
	for _, p := range perms {
		if p.Path == "*" {
			return true
		}
		if p.Method != "*" && p.Method != method {
			continue
		}
		if matchPath(p.Path, reqPath) {
			return true
		}
	}
	return false
}

func matchPath(pattern, reqPath string) bool {
	if strings.HasSuffix(pattern, "/**") {
		prefix := strings.TrimSuffix(pattern, "**")
		return reqPath == strings.TrimSuffix(prefix, "/") || strings.HasPrefix(reqPath, prefix)
	}

	patternParts := strings.Split(pattern, "/")
	pathParts := strings.Split(reqPath, "/")
	if len(patternParts) != len(pathParts) {
		return false
	}
	for i := range patternParts {
		if patternParts[i] == "*" {
			continue
		}
		if patternParts[i] != pathParts[i] {
			return false
		}
	}
	return true
}
