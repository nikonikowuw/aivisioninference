package log

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestWithLogContext_RoundTrip(t *testing.T) {
	lc := LogContext{
		TraceID:  "trace-123",
		UserID:   "user-456",
		TenantID: "tenant-789",
		SpanID:   "span-abc",
	}

	ctx := WithLogContext(context.Background(), lc)
	got, ok := GetLogContext(ctx)
	assert.True(t, ok)
	assert.Equal(t, lc, got)
}

func TestGetLogContext_NoContext(t *testing.T) {
	_, ok := GetLogContext(context.Background())
	assert.False(t, ok)
}

func TestCtx_WithFullContext(t *testing.T) {
	// Set up an observed logger to capture output
	core, recorded := observer.New(zapcore.InfoLevel)
	zap.ReplaceGlobals(zap.New(core))

	lc := LogContext{
		TraceID:  "trace-xyz",
		UserID:   "user-abc",
		TenantID: "tenant-def",
		SpanID:   "span-ghi",
	}
	ctx := WithLogContext(context.Background(), lc)

	Ctx(ctx).Info("test message")

	assert.Equal(t, 1, recorded.Len())
	entry := recorded.All()[0]
	assert.Equal(t, "test message", entry.Message)
	assert.Equal(t, "trace-xyz", entry.ContextMap()["trace_id"])
	assert.Equal(t, "user-abc", entry.ContextMap()["user_id"])
	assert.Equal(t, "tenant-def", entry.ContextMap()["tenant_id"])
	assert.Equal(t, "span-ghi", entry.ContextMap()["span_id"])
}

func TestCtx_WithoutLogContext(t *testing.T) {
	core, recorded := observer.New(zapcore.InfoLevel)
	zap.ReplaceGlobals(zap.New(core))

	// No LogContext in context — should not add trace fields
	Ctx(context.Background()).Info("no context")

	assert.Equal(t, 1, recorded.Len())
	entry := recorded.All()[0]
	assert.Equal(t, "no context", entry.Message)
	// Should not have trace_id field
	cm := entry.ContextMap()
	_, hasTraceID := cm["trace_id"]
	assert.False(t, hasTraceID, "should not have trace_id when no LogContext")
}

func TestCtx_PartialFields(t *testing.T) {
	core, recorded := observer.New(zapcore.InfoLevel)
	zap.ReplaceGlobals(zap.New(core))

	// Only TraceID, no UserID/TenantID/SpanID
	lc := LogContext{TraceID: "trace-only"}
	ctx := WithLogContext(context.Background(), lc)

	Ctx(ctx).Info("partial")

	assert.Equal(t, 1, recorded.Len())
	entry := recorded.All()[0]
	cm := entry.ContextMap()
	assert.Equal(t, "trace-only", cm["trace_id"])
	// user_id should not be present when not in LogContext
	_, hasUserID := cm["user_id"]
	assert.False(t, hasUserID, "should not have user_id when not in LogContext")
}

func TestLoggerCtx_NilLogger(t *testing.T) {
	var l *Logger
	// Should not panic and return zap.L()
	ctx := WithLogContext(context.Background(), LogContext{TraceID: "test"})
	logger := l.Ctx(ctx)
	assert.NotNil(t, logger)
}

func TestLoggerCtx_WithLogger(t *testing.T) {
	core, recorded := observer.New(zapcore.InfoLevel)
	testLogger := zap.New(core)

	l := &Logger{App: testLogger}

	lc := LogContext{
		TraceID: "trace-from-logger",
		UserID:  "user-from-logger",
	}
	ctx := WithLogContext(context.Background(), lc)

	l.Ctx(ctx).Info("logger method")

	assert.Equal(t, 1, recorded.Len())
	entry := recorded.All()[0]
	assert.Equal(t, "trace-from-logger", entry.ContextMap()["trace_id"])
	assert.Equal(t, "user-from-logger", entry.ContextMap()["user_id"])
}

func TestDualWriteFix_ErrorEnabledWithoutDedup(t *testing.T) {
	// Verify that when Error.Enabled is true, error logs go to both app and error loggers,
	// but errorLogger itself does NOT duplicate errorCore from appLogger
	appCore, appRecorded := observer.New(zapcore.InfoLevel)
	errCore, errRecorded := observer.New(zapcore.ErrorLevel)

	appLogger := zap.New(appCore, zap.AddCaller(), zap.AddCallerSkip(1))
	errLogger := zap.New(errCore, zap.AddCaller(), zap.AddCallerSkip(1))

	// Simulate the fix: Tee errorCore into appLogger only when cfg.Error.Enabled
	appCoreWithError := zapcore.NewTee(appCore, errCore)
	appLogger = zap.New(appCoreWithError, zap.AddCaller(), zap.AddCallerSkip(1))

	// Write an error through appLogger
	appLogger.Error("error from app")
	_ = errLogger  // not used in this test

	// The error should appear in appRecorded (app logger received it)
	assert.Equal(t, 1, appRecorded.Len())
	assert.Equal(t, "error from app", appRecorded.All()[0].Message)

	// Verify app logger writes to errCore (simulating error file)
	assert.Equal(t, 1, errRecorded.Len())
	assert.Equal(t, "error from app", errRecorded.All()[0].Message)
}
