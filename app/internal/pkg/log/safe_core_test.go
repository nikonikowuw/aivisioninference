package log

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestSafeCore_SanitizesSensitiveKeys(t *testing.T) {
	core, recorded := observer.New(zapcore.InfoLevel)
	safe := NewSafeCore(core, []string{"password", "token"})

	logger := zap.New(safe)

	logger.Info("test",
		zap.String("password", "supersecret"),
		zap.String("username", "admin"),
		zap.String("token", "eyJhbGciOiJIUzI1NiJ9"),
	)

	assert.Equal(t, 1, recorded.Len())
	entry := recorded.All()[0]

	cm := entry.ContextMap()
	assert.Equal(t, "****", cm["password"], "password should be masked")
	assert.Equal(t, "admin", cm["username"], "username should stay intact")
	assert.Equal(t, "****", cm["token"], "token should be masked")
}

func TestSafeCore_CustomKeys(t *testing.T) {
	core, recorded := observer.New(zapcore.InfoLevel)
	safe := NewSafeCore(core, []string{"api_key", "email"})

	logger := zap.New(safe)

	logger.Info("test",
		zap.String("api_key", "sk-123456"),
		zap.String("email", "user@example.com"),
		zap.String("password", "should-not-be-masked"),
	)

	assert.Equal(t, 1, recorded.Len())
	entry := recorded.All()[0]

	cm := entry.ContextMap()
	assert.Equal(t, "****", cm["api_key"], "api_key should be masked")
	assert.Equal(t, "****", cm["email"], "email should be masked")
	assert.Equal(t, "should-not-be-masked", cm["password"], "password not in custom keys list")
}

func TestSafeCore_NonStringFieldsUntouched(t *testing.T) {
	core, recorded := observer.New(zapcore.InfoLevel)
	safe := NewSafeCore(core, []string{"count"})

	logger := zap.New(safe)

	logger.Info("test",
		zap.Int("count", 42),
		zap.Float64("price", 19.99),
	)

	assert.Equal(t, 1, recorded.Len())
	entry := recorded.All()[0]

	cm := entry.ContextMap()
	// Int field named "count" gets sanitized (type converted to string "****")
	assert.Equal(t, "****", cm["count"], "int field with sensitive key should be masked")
	assert.Equal(t, float64(19.99), cm["price"], "non-sensitive float should stay intact")
}

func TestSafeCore_DisabledNoOp(t *testing.T) {
	// When sanitize is not enabled, core should not be wrapped
	core, recorded := observer.New(zapcore.InfoLevel)

	logger := zap.New(core)

	logger.Info("test",
		zap.String("password", "visible"),
	)

	assert.Equal(t, 1, recorded.Len())
	entry := recorded.All()[0]
	assert.Equal(t, "visible", entry.ContextMap()["password"], "password should be visible when no SafeCore")
}

func TestSetLevel_Valid(t *testing.T) {
	l := &Logger{level: zap.NewAtomicLevel()}

	err := l.SetLevel("debug")
	assert.NoError(t, err)
	assert.Equal(t, "debug", l.level.String())

	err = l.SetLevel("error")
	assert.NoError(t, err)
	assert.Equal(t, "error", l.level.String())
}

func TestSetLevel_Invalid(t *testing.T) {
	l := &Logger{level: zap.NewAtomicLevel()}

	// SetLevel now validates and returns error for unrecognized levels
	err := l.SetLevel("invalid")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unrecognized level")
	// level should stay at default info
	assert.Equal(t, "info", l.level.String())
}

func TestAtomicLevel(t *testing.T) {
	level := zap.NewAtomicLevelAt(zap.WarnLevel)
	l := &Logger{level: level}

	// AtomicLevel 返回指针，应指向 Logger 内部的 level 字段
	assert.Equal(t, &level, l.AtomicLevel())
	assert.Equal(t, "warn", l.AtomicLevel().String())
}

func TestSafeCore_CaseInsensitiveKeyMatch(t *testing.T) {
	core, recorded := observer.New(zapcore.InfoLevel)
	safe := NewSafeCore(core, []string{"Authorization", "Session-Id"})

	logger := zap.New(safe)

	logger.Info("test",
		zap.String("authorization", "Bearer token123"),
		zap.String("SESSION-ID", "abc123"),
		zap.String("normal_field", "visible"),
	)

	assert.Equal(t, 1, recorded.Len())
	entry := recorded.All()[0]

	cm := entry.ContextMap()
	assert.Equal(t, "****", cm["authorization"], "authorization (lowercase key) should be masked")
	assert.Equal(t, "****", cm["SESSION-ID"], "SESSION-ID (uppercase field) should be masked")
	assert.Equal(t, "visible", cm["normal_field"], "non-sensitive field should stay intact")
}
