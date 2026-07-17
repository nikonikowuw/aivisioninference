package log

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestHookCore_FiresOnMinLevel(t *testing.T) {
	core, recorded := observer.New(zapcore.InfoLevel)

	var received []ErrorEntry
	hook := ErrorHookFunc(func(entry ErrorEntry) {
		received = append(received, entry)
	})

	hookCore := NewHookCore(core, zapcore.WarnLevel, []ErrorHook{hook})
	logger := zap.New(hookCore)

	logger.Info("should not fire")    // below WarnLevel
	logger.Warn("should fire warn")   // at WarnLevel
	logger.Error("should fire error") // above WarnLevel

	assert.Equal(t, 3, recorded.Len())
	assert.Len(t, received, 2, "warn level and above should fire, info should not")
	if len(received) >= 2 {
		assert.Equal(t, "should fire warn", received[0].Message)
		assert.Equal(t, "warn", received[0].Level)
		assert.Equal(t, "should fire error", received[1].Message)
		assert.Equal(t, "error", received[1].Level)
	}
}

func TestHookCore_NoHooks(t *testing.T) {
	core, recorded := observer.New(zapcore.InfoLevel)

	hookCore := NewHookCore(core, zapcore.ErrorLevel, nil)
	logger := zap.New(hookCore)

	logger.Error("no hooks registered")
	assert.Equal(t, 1, recorded.Len())
	// Should not panic
}

func TestWebhookErrorHook_OnError(t *testing.T) {
	// WebhookErrorHook just fires an async POST — verify it doesn't panic
	hook := NewWebhookErrorHook("http://127.0.0.1:1/nonexistent")
	entry := ErrorEntry{
		Time:    time.Now(),
		Message: "test",
		Level:   "error",
	}

	// Should not panic even with an unreachable URL (async + fire-and-forget)
	hook.OnError(entry)
	time.Sleep(10 * time.Millisecond)
}

func TestRegisterErrorHook(t *testing.T) {
	hook := ErrorHookFunc(func(entry ErrorEntry) {})

	logger := &Logger{}
	logger.RegisterErrorHook(hook)

	assert.Len(t, logger.hooks, 1)

	logger.RegisterErrorHook(hook)
	assert.Len(t, logger.hooks, 2)
}

func TestCaptureStack(t *testing.T) {
	stack := captureStack(1)
	assert.NotEmpty(t, stack)
	assert.Contains(t, stack, "TestCaptureStack") // should contain this function name
}
