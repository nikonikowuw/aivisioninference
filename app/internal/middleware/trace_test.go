package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	applog "github.com/niko-admin/niko-admin/internal/pkg/log"
)

func TestTraceID_GeneratesNewID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(TraceID())
	r.GET("/test", func(c *gin.Context) {
		traceID, _ := c.Get(ContextKeyTraceID)
		c.String(http.StatusOK, traceID.(string))
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotEmpty(t, w.Body.String())
	assert.NotEmpty(t, w.Header().Get("X-Request-ID"))
	assert.Len(t, w.Body.String(), 36) // UUID v4 length
}

func TestTraceID_ReadsXTraceID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(TraceID())
	r.GET("/test", func(c *gin.Context) {
		traceID, _ := c.Get(ContextKeyTraceID)
		lc, ok := applog.GetLogContext(c.Request.Context())
		require.True(t, ok)
		c.String(http.StatusOK, traceID.(string)+"|"+lc.TraceID)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Trace-ID", "custom-trace-123")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "custom-trace-123|custom-trace-123", w.Body.String())
	assert.Equal(t, "custom-trace-123", w.Header().Get("X-Request-ID"))
}

func TestTraceID_FallsBackToXRequestID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(TraceID())
	r.GET("/test", func(c *gin.Context) {
		traceID, _ := c.Get(ContextKeyTraceID)
		c.String(http.StatusOK, traceID.(string))
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Request-ID", "req-id-from-client")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "req-id-from-client", w.Body.String())
}

func TestTraceID_InjectLogContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(TraceID())
	r.GET("/test", func(c *gin.Context) {
		lc, ok := applog.GetLogContext(c.Request.Context())
		require.True(t, ok)
		assert.NotEmpty(t, lc.TraceID)
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTraceID_OrderRespectsLogger(t *testing.T) {
	gin.SetMode(gin.TestMode)

	accessLogger := zap.NewNop()

	r := gin.New()
	r.Use(TraceID())
	r.Use(Logger(accessLogger))
	r.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotEmpty(t, w.Header().Get("X-Request-ID"))
}

func TestGinTraceFields_WithLogContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(TraceID())
	r.GET("/test", func(c *gin.Context) {
		fields := ginTraceFields(c)
		// trace_id should be present, user_id absent (Auth hasn't run)
		require.Len(t, fields, 1)
		assert.Equal(t, "trace_id", fields[0].Key)
		assert.NotEmpty(t, fields[0].String)
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGinTraceFields_WithUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(TraceID())
	r.GET("/test", func(c *gin.Context) {
		// Simulate Auth middleware setting user_id
		c.Set(ContextKeyUserID, "user-123")

		fields := ginTraceFields(c)
		require.Len(t, fields, 2)
		assert.Equal(t, "trace_id", fields[0].Key)
		assert.Equal(t, "user_id", fields[1].Key)
		assert.Equal(t, "user-123", fields[1].String)
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGinTraceFields_EmptyUserIDOmitted(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(TraceID())
	r.GET("/test", func(c *gin.Context) {
		// Set user_id to empty string — should be omitted
		c.Set(ContextKeyUserID, "")

		fields := ginTraceFields(c)
		require.Len(t, fields, 1)
		assert.Equal(t, "trace_id", fields[0].Key)
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGinTraceFields_EmptyTraceIDOmitted(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(TraceID())
	r.GET("/test", func(c *gin.Context) {
		// Clear the LogContext to simulate no trace
		ctx := applog.WithLogContext(c.Request.Context(), applog.LogContext{})
		c.Request = c.Request.WithContext(ctx)

		fields := ginTraceFields(c)
		// Both empty — no fields returned
		assert.Empty(t, fields)
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
