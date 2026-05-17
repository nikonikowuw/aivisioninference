package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/niko-admin/niko-admin/internal/model"
	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

type fakeAuditLogger struct {
	createFn func(ctx context.Context, log *model.AuditLog) error
	logs     []*model.AuditLog
}

// Create 记录审计日志写入参数，用于验证中间件行为。
func (f *fakeAuditLogger) Create(ctx context.Context, log *model.AuditLog) error {
	if f.createFn != nil {
		if err := f.createFn(ctx, log); err != nil {
			return err
		}
	}
	f.logs = append(f.logs, log)
	return nil
}

func TestAudit_SkipPathsAndMethods(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name   string
		method string
		path   string
	}{
		{name: "skip options", method: http.MethodOptions, path: "/api/v1/users"},
		{name: "skip health", method: http.MethodGet, path: "/health"},
		{name: "skip swagger", method: http.MethodGet, path: "/swagger/index.html"},
		{name: "skip static", method: http.MethodGet, path: "/static/app.js"},
		{name: "skip auth login", method: http.MethodPost, path: "/api/v1/auth/login"},
		{name: "skip auth refresh", method: http.MethodPost, path: "/api/v1/auth/refresh"},
		{name: "skip audit list", method: http.MethodGet, path: "/api/v1/audit-logs"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			recorderSvc := &fakeAuditLogger{}
			r := gin.New()
			r.Use(Audit(recorderSvc))
			r.Handle(tc.method, tc.path, func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"ok": true})
			})

			req := httptest.NewRequest(tc.method, tc.path, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			require.Equal(t, http.StatusOK, w.Code)
			require.Empty(t, recorderSvc.logs)
		})
	}
}

func TestAudit_RecordRequestMetadataAndSanitizedSummaries(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorderSvc := &fakeAuditLogger{}
	r := gin.New()
	r.Use(Audit(recorderSvc))
	r.POST("/api/v1/users", func(c *gin.Context) {
		require.NotNil(t, c.Error(errors.New("db connection timeout")))
		c.JSON(http.StatusInternalServerError, gin.H{"code": 50000, "message": "内部错误"})
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/users?token=abc", strings.NewReader(`{"password":"secret","name":"niko"}`))
	req.Header.Set("User-Agent", "unit-test-agent")
	req.Header.Set("Authorization", "Bearer secret-token")
	req.Header.Set("Cookie", "sid=secret")
	req.RemoteAddr = "10.20.30.40:5678"

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
	require.Len(t, recorderSvc.logs, 1)
	logged := recorderSvc.logs[0]
	require.Equal(t, http.MethodPost, logged.RequestMethod)
	require.Equal(t, "/api/v1/users", logged.RequestPath)
	require.Equal(t, "unit-test-agent", logged.UserAgent)
	require.Equal(t, "10.20.30.40", logged.RequestIP)
	require.Equal(t, http.StatusInternalServerError, logged.ResponseStatus)
	require.GreaterOrEqual(t, logged.DurationMs, int64(0))
	require.Equal(t, "users", logged.ResourceType)
	require.Equal(t, "failed", logged.ResultSummary)
	require.Empty(t, logged.RequestBody)
	serialized := strings.ToLower(logged.UserAgent + logged.ResultSummary + logged.RequestBody)
	require.NotContains(t, serialized, "password")
	require.NotContains(t, serialized, "secret")
	require.NotContains(t, serialized, "bearer")
	require.NotContains(t, serialized, "cookie")
	require.NotContains(t, serialized, "db connection")
}

func TestAudit_RecordsAuthenticatedUserIdentity(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorderSvc := &fakeAuditLogger{}
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(ContextKeyUserID, "user-123")
		c.Set(ContextKeyUsername, "niko")
		c.Next()
	})
	r.Use(Audit(recorderSvc))
	r.POST("/api/v1/users", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Len(t, recorderSvc.logs, 1)
	require.NotNil(t, recorderSvc.logs[0].UserID)
	require.Equal(t, "user-123", *recorderSvc.logs[0].UserID)
	require.Equal(t, "niko", recorderSvc.logs[0].Username)
}

func TestTruncateAuditSummary_PreservesUTF8(t *testing.T) {
	truncated := TruncateAuditSummary("你好世界", 3)

	require.True(t, utf8.ValidString(truncated))
	require.Equal(t, "你好世", truncated)
}

func TestAudit_FailToWriteDoesNotAffectBusinessResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorderSvc := &fakeAuditLogger{
		createFn: func(ctx context.Context, log *model.AuditLog) error {
			return errors.New("insert failed")
		},
	}

	r := gin.New()
	r.Use(Audit(recorderSvc))
	r.GET("/api/v1/users", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Empty(t, recorderSvc.logs)
}

func TestAudit_InfersAppErrorStatusBeforeErrorHandlerWritesResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorderSvc := &fakeAuditLogger{}
	r := gin.New()
	r.Use(Audit(recorderSvc))
	r.GET("/api/v1/users", func(c *gin.Context) {
		require.NotNil(t, c.Error(apperrors.New(apperrors.ErrForbidden, "无权访问")))
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Len(t, recorderSvc.logs, 1)
	require.Equal(t, http.StatusForbidden, recorderSvc.logs[0].ResponseStatus)
	require.Equal(t, "failed", recorderSvc.logs[0].ResultSummary)
}

func TestAudit_InfersWrappedAppErrorStatusBeforeErrorHandlerWritesResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorderSvc := &fakeAuditLogger{}
	r := gin.New()
	r.Use(Audit(recorderSvc))
	r.GET("/api/v1/users", func(c *gin.Context) {
		wrappedErr := errors.New("outer")
		require.NotNil(t, c.Error(errors.Join(wrappedErr, apperrors.New(apperrors.ErrForbidden, "无权访问"))))
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Len(t, recorderSvc.logs, 1)
	require.Equal(t, http.StatusForbidden, recorderSvc.logs[0].ResponseStatus)
	require.Equal(t, "failed", recorderSvc.logs[0].ResultSummary)
}
