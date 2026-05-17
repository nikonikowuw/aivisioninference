package router

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/niko-admin/niko-admin/internal/middleware"
	"github.com/niko-admin/niko-admin/internal/model"
	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
)

type routerAuditRecorder struct {
	logs []*model.AuditLog
}

// Create 记录路由中间件链路产生的审计日志，避免测试依赖数据库。
func (r *routerAuditRecorder) Create(ctx context.Context, log *model.AuditLog) error {
	r.logs = append(r.logs, log)
	return nil
}

func TestProtectedMiddlewareChainSkipsAuditWhenAuthAborts(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := &routerAuditRecorder{}
	r := gin.New()
	protected := r.Group("/api/v1")
	protected.Use(func(c *gin.Context) {
		require.NotNil(t, c.Error(apperrors.New(apperrors.ErrUnauthorized, "缺少认证令牌")))
		c.Abort()
	})
	protected.Use(middleware.Audit(recorder))
	protected.GET("/users", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Empty(t, recorder.logs)
}

func TestProtectedMiddlewareChainAuditsAuthenticatedRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := &routerAuditRecorder{}
	r := gin.New()
	protected := r.Group("/api/v1")
	protected.Use(func(c *gin.Context) {
		c.Set(middleware.ContextKeyUserID, "user-123")
		c.Set(middleware.ContextKeyUsername, "niko")
		c.Next()
	})
	protected.Use(middleware.Audit(recorder))
	protected.GET("/users", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Len(t, recorder.logs, 1)
	require.NotNil(t, recorder.logs[0].UserID)
	require.Equal(t, "user-123", *recorder.logs[0].UserID)
	require.Equal(t, "niko", recorder.logs[0].Username)
	require.Equal(t, http.StatusOK, recorder.logs[0].ResponseStatus)
}
