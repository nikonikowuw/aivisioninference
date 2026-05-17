package log

import (
	"testing"

	"github.com/niko-admin/niko-admin/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestParseLevel(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected any
	}{
		{"debug", "debug", "debug"},
		{"info", "info", "info"},
		{"warn", "warn", "warn"},
		{"warning", "warning", "warn"},
		{"error", "error", "error"},
		{"dpanic", "dpanic", "dpanic"},
		{"panic", "panic", "panic"},
		{"fatal", "fatal", "fatal"},
		{"uppercase INFO", "INFO", "info"},
		{"mixed case Warn", "Warn", "warn"},
		{"invalid falls back to info", "invalid", "info"},
		{"empty falls back to info", "", "info"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseLevel(tt.input)
			assert.Equal(t, tt.expected, result.String())
		})
	}
}

func TestInit_WithDisabledFileLogging(t *testing.T) {
	cfg := config.LogConfig{
		Level:  "info",
		Format: "console",
		Output: "stdout",
		Access: config.LogFileConfig{Enabled: false},
		App:    config.LogFileConfig{Enabled: false},
		Error:  config.LogFileConfig{Enabled: false},
	}

	logger := Init(cfg)
	defer logger.Sync()

	assert.NotNil(t, logger)
	assert.NotNil(t, logger.Access)
	assert.NotNil(t, logger.App)
	assert.NotNil(t, logger.Error)
}

func TestInit_WithEnabledFileLogging(t *testing.T) {
	cfg := config.LogConfig{
		Level:  "debug",
		Format: "json",
		Output: "both",
		Access: config.LogFileConfig{
			Enabled:    true,
			Path:       t.TempDir() + "/access.log",
			MaxSize:    100,
			MaxBackups: 3,
			MaxAge:     7,
			Compress:   false,
		},
		App: config.LogFileConfig{
			Enabled:    true,
			Path:       t.TempDir() + "/app.log",
			MaxSize:    100,
			MaxBackups: 3,
			MaxAge:     7,
			Compress:   false,
		},
		Error: config.LogFileConfig{
			Enabled:    true,
			Path:       t.TempDir() + "/error.log",
			MaxSize:    50,
			MaxBackups: 5,
			MaxAge:     30,
			Compress:   true,
		},
	}

	logger := Init(cfg)
	defer logger.Sync()

	assert.NotNil(t, logger)
	assert.NotNil(t, logger.Access)
	assert.NotNil(t, logger.App)
	assert.NotNil(t, logger.Error)

	// Verify logging works without panic
	logger.Access.Info("test access log")
	logger.App.Info("test app log")
	logger.Error.Error("test error log")
}

func TestSync_NilLogger(t *testing.T) {
	var logger *Logger
	// Should not panic
	logger.Sync()
}

func TestInit_SetsGlobalLogger(t *testing.T) {
	cfg := config.LogConfig{
		Level:  "warn",
		Format: "console",
		Output: "stdout",
		Access: config.LogFileConfig{Enabled: false},
		App:    config.LogFileConfig{Enabled: false},
		Error:  config.LogFileConfig{Enabled: false},
	}

	logger := Init(cfg)
	defer logger.Sync()

	// Global logger should be set
	assert.NotNil(t, logger.App)
}
