// Package log provides centralized logging initialization with zap and lumberjack.
// It supports splitting logs by function (access/app/error) with automatic rotation.
package log

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/niko-admin/niko-admin/internal/config"
)

// Logger holds the application loggers for different purposes.
// level 字段所有应用日志 core 共享指针，运行时调级对所有 core 生效。
type Logger struct {
	Access *zap.Logger // HTTP request logs
	App    *zap.Logger // All application logs
	Error  *zap.Logger // Error+ level logs
	level  zap.AtomicLevel
	hooks  []ErrorHook  // 错误日志告警回调
}

// Ctx 返回一个带有全链路追踪字段的 App Logger。
// 当 context 中不存在 LogContext 时，返回 App Logger 自身。
func (l *Logger) Ctx(ctx context.Context) *zap.Logger {
	if l == nil || l.App == nil {
		return zap.L()
	}
	return addContextFields(l.App, ctx)
}

// Init initializes the logging system based on the provided configuration.
// It creates three loggers (access/app/error) with optional file output and rotation.
// 所有应用日志 core 共享 logger.level 的指针，SetLevel 调级即时生效。
func Init(cfg config.LogConfig) *Logger {
	logger := &Logger{
		level: parseLevel(cfg.Level),
	}

	// Build encoders
	fileEncoder := buildFileEncoder(cfg.Format)
	stdoutEncoder := buildConsoleEncoder(cfg.Format)

	// Build cores: 传递 &logger.level 使所有 core 共享同一 AtomicLevel 实例
	// 运行时通过 SetLevel 调级对所有 core 即时生效
	accessCore := buildCore(cfg.Access, &logger.level, fileEncoder, stdoutEncoder, cfg.Output)
	appCore := buildCore(cfg.App, &logger.level, fileEncoder, stdoutEncoder, cfg.Output)

	// errorCore 使用独立的固定 Error 级别，不受运行时调级影响
	errorLevel := zap.NewAtomicLevelAt(zap.ErrorLevel)
	errorCore := buildCore(cfg.Error, &errorLevel, fileEncoder, stdoutEncoder, cfg.Output)

	// Wrap cores with SafeCore for sensitive data sanitization
	if cfg.Sanitize.Enabled {
		sanitizeKeys := cfg.Sanitize.Keys
		accessCore = NewSafeCore(accessCore, sanitizeKeys)
		appCore = NewSafeCore(appCore, sanitizeKeys)
		errorCore = NewSafeCore(errorCore, sanitizeKeys)
	}

	// Apply sampling when configured
	if cfg.Sampling.Enabled {
		accessCore = wrapWithSampling(accessCore, cfg.Sampling)
		appCore = wrapWithSampling(appCore, cfg.Sampling)
		errorCore = wrapWithSampling(errorCore, cfg.Sampling)
	}

	// Wrap with HookCore for error-level alert callbacks
	if cfg.ErrorHook.Enabled {
		minLevel := zapcore.ErrorLevel
		switch strings.ToLower(cfg.ErrorHook.MinLevel) {
		case "warn", "warning":
			minLevel = zapcore.WarnLevel
		case "error":
			minLevel = zapcore.ErrorLevel
		case "dpanic":
			minLevel = zapcore.DPanicLevel
		}

		// Create the webhook hook if URL is configured
		var hooks []ErrorHook
		if cfg.ErrorHook.URL != "" {
			hooks = append(hooks, NewWebhookErrorHook(cfg.ErrorHook.URL))
		}
		logger.hooks = hooks

		accessCore = NewHookCore(accessCore, minLevel, hooks)
		appCore = NewHookCore(appCore, minLevel, hooks)
		errorCore = NewHookCore(errorCore, minLevel, hooks)
	}

	// Compose: error core 不默认混入 app core，避免 error 级别日志双写
	// 当 cfg.Error.Enabled 时，errorCore 追加到 appLogger 使其同时写入错误文件
	accessLogger := zap.New(accessCore, zap.AddCaller(), zap.AddCallerSkip(1))
	appLogger := zap.New(appCore, zap.AddCaller(), zap.AddCallerSkip(1))
	errorLogger := zap.New(errorCore, zap.AddCaller(), zap.AddCallerSkip(1))

	// 仅在配置了独立错误文件时，将 errorCore 加入 appLogger
	if cfg.Error.Enabled {
		appCoreWithError := zapcore.NewTee(appCore, errorCore)
		appLogger = zap.New(appCoreWithError, zap.AddCaller(), zap.AddCallerSkip(1))
	}

	// Set global logger
	zap.ReplaceGlobals(appLogger)

	logger.Access = accessLogger
	logger.App = appLogger
	logger.Error = errorLogger
	return logger
}

// RegisterErrorHook 注册一个错误日志告警回调。
// 调用后，所有 >= Error 级别的日志条目会通过 hook 异步通知。
func (l *Logger) RegisterErrorHook(hook ErrorHook) {
	if l == nil {
		return
	}
	l.hooks = append(l.hooks, hook)
}

// buildCore creates a zapcore.Core with separate file and stdout writers.
// In "both" mode, file gets clean output (no ANSI colors) while stdout gets colored output.
// level 接受 *zap.AtomicLevel，zapcore.NewCore 将其作为 LevelEnabler 接口存储指针。
// 所有调用方共享同一 AtomicLevel 实例，运行时调级对所有 core 生效。
func buildCore(fileCfg config.LogFileConfig, level *zap.AtomicLevel, fileEncoder, stdoutEncoder zapcore.Encoder, output string) zapcore.Core {
	fileSyncer := newFileWriter(fileCfg)
	stdoutSyncer := zapcore.Lock(os.Stdout)

	switch output {
	case "file":
		return zapcore.NewCore(fileEncoder, fileSyncer, level)
	case "stdout":
		return zapcore.NewCore(stdoutEncoder, stdoutSyncer, level)
	default: // "both"
		return zapcore.NewTee(
			zapcore.NewCore(fileEncoder, fileSyncer, level),
			zapcore.NewCore(stdoutEncoder, stdoutSyncer, level),
		)
	}
}

// Sync flushes any buffered log entries for all loggers.
func (l *Logger) Sync() {
	if l == nil {
		return
	}
	_ = l.Access.Sync()
	_ = l.App.Sync()
	_ = l.Error.Sync()
}

// parseLevel converts a string level to zap.AtomicLevel.
// Supports: debug, info, warn/warning, error, dpanic, panic, fatal.
// Falls back to info for unrecognized values.
func parseLevel(s string) zap.AtomicLevel {
	switch strings.ToLower(s) {
	case "debug":
		return zap.NewAtomicLevelAt(zap.DebugLevel)
	case "info":
		return zap.NewAtomicLevelAt(zap.InfoLevel)
	case "warn", "warning":
		return zap.NewAtomicLevelAt(zap.WarnLevel)
	case "error":
		return zap.NewAtomicLevelAt(zap.ErrorLevel)
	case "dpanic":
		return zap.NewAtomicLevelAt(zap.DPanicLevel)
	case "panic":
		return zap.NewAtomicLevelAt(zap.PanicLevel)
	case "fatal":
		return zap.NewAtomicLevelAt(zap.FatalLevel)
	default:
		return zap.NewAtomicLevelAt(zap.InfoLevel)
	}
}

// newFileWriter creates a WriteSyncer for log file output.
// When cfg.TimeBased is true, it returns a dailyRotateSyncer that splits
// logs by date with date-named files. Otherwise, it returns a lumberjack-backed
// WriteSyncer with size-based rotation.
// newFileWriter creates a WriteSyncer for log file output, optionally wrapped with
// BufferedWriteSyncer for performant buffered writes. Supports both lumberjack
// (size-based) and dailyRotateSyncer (date-based) rotation backends.
// Returns a no-op WriteSyncer if file logging is disabled.
func newFileWriter(cfg config.LogFileConfig) zapcore.WriteSyncer {
	if !cfg.Enabled || cfg.Path == "" {
		return zapcore.AddSync(io.Discard)
	}

	// Build the base file syncer
	var fileWriter io.Writer
	if cfg.TimeBased {
		// Time-based (daily) rotation with date-named files
		fileWriter = newDailyRotateSyncer(cfg)
	} else {
		// Size-based rotation via lumberjack
		dir := filepath.Dir(cfg.Path)
		if dir != "" {
			_ = os.MkdirAll(dir, 0o755)
		}
		fileWriter = &lumberjack.Logger{
			Filename:   cfg.Path,
			MaxSize:    cfg.MaxSize,
			MaxBackups: cfg.MaxBackups,
			MaxAge:     cfg.MaxAge,
			Compress:   cfg.Compress,
		}
	}

	ws := zapcore.AddSync(fileWriter)

	// Wrap with BufferedWriteSyncer when buffer size is configured
	if cfg.BufferSize > 0 {
		flushInterval := cfg.FlushInterval
		if flushInterval <= 0 {
			flushInterval = 5 * time.Second
		}
		ws = &zapcore.BufferedWriteSyncer{
			WS:            ws,
			Size:          cfg.BufferSize,
			FlushInterval: flushInterval,
		}
	}

	return ws
}

// wrapWithSampling wraps a core with zap's sampling filter when enabled.
func wrapWithSampling(core zapcore.Core, cfg config.LogSamplingConfig) zapcore.Core {
	tickInterval := time.Duration(cfg.TickInterval) * time.Second
	if tickInterval <= 0 {
		tickInterval = time.Second
	}
	initial := cfg.Initial
	if initial <= 0 {
		initial = 100
	}
	thereafter := cfg.Thereafter
	if thereafter <= 0 {
		thereafter = 100
	}
	return zapcore.NewSamplerWithOptions(
		core,
		tickInterval,
		initial,
		thereafter,
	)
}

// SetLevel 动态设置日志级别。支持: debug, info, warn, error, dpanic, panic, fatal。
// 仅接受 zap.UnmarshalText 识别的规范级别名（不含 "warning"）。
func (l *Logger) SetLevel(level string) error {
	if err := l.level.UnmarshalText([]byte(level)); err != nil {
		return fmt.Errorf("invalid log level %q: %w", level, err)
	}
	return nil
}

// AtomicLevel 返回当前日志级别的指针。所有应用日志 core 共享此实例，
// 通过返回的指针修改级别对所有 core 即时生效。
func (l *Logger) AtomicLevel() *zap.AtomicLevel {
	return &l.level
}

// buildFileEncoder creates an encoder for file output (no ANSI colors).
func buildFileEncoder(format string) zapcore.Encoder {
	if format == "json" {
		cfg := zap.NewProductionEncoderConfig()
		cfg.EncodeTime = zapcore.ISO8601TimeEncoder
		cfg.EncodeLevel = zapcore.LowercaseLevelEncoder
		cfg.EncodeDuration = zapcore.MillisDurationEncoder
		cfg.EncodeCaller = zapcore.ShortCallerEncoder
		return zapcore.NewJSONEncoder(cfg)
	}
	cfg := zap.NewDevelopmentEncoderConfig()
	cfg.EncodeLevel = zapcore.CapitalLevelEncoder
	cfg.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05")
	cfg.EncodeDuration = zapcore.MillisDurationEncoder
	cfg.EncodeCaller = zapcore.ShortCallerEncoder
	return zapcore.NewConsoleEncoder(cfg)
}

// buildConsoleEncoder creates an encoder for stdout output (with ANSI colors in console mode).
func buildConsoleEncoder(format string) zapcore.Encoder {
	if format == "json" {
		cfg := zap.NewProductionEncoderConfig()
		cfg.EncodeTime = zapcore.ISO8601TimeEncoder
		cfg.EncodeLevel = zapcore.LowercaseLevelEncoder
		cfg.EncodeDuration = zapcore.MillisDurationEncoder
		cfg.EncodeCaller = zapcore.ShortCallerEncoder
		return zapcore.NewJSONEncoder(cfg)
	}
	cfg := zap.NewDevelopmentEncoderConfig()
	cfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
	cfg.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05")
	cfg.EncodeDuration = zapcore.MillisDurationEncoder
	cfg.EncodeCaller = zapcore.ShortCallerEncoder
	return zapcore.NewConsoleEncoder(cfg)
}
