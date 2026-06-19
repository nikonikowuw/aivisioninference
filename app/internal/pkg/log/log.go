// Package log provides centralized logging initialization with zap and lumberjack.
// It supports splitting logs by function (access/app/error) with automatic rotation.
package log

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/niko-admin/niko-admin/internal/config"
)

// Logger holds the application loggers for different purposes.
type Logger struct {
	Access *zap.Logger // HTTP request logs
	App    *zap.Logger // All application logs
	Error  *zap.Logger // Error+ level logs
	level  zap.AtomicLevel
}

// Init initializes the logging system based on the provided configuration.
// It creates three loggers (access/app/error) with optional file output and rotation.
func Init(cfg config.LogConfig) *Logger {
	level := parseLevel(cfg.Level)

	// Build encoders
	fileEncoder := buildFileEncoder(cfg.Format)
	stdoutEncoder := buildConsoleEncoder(cfg.Format)

	// Build cores: each logger gets separate file and stdout cores
	// to avoid ANSI color codes leaking into log files
	accessCore := buildCore(cfg.Access, level, fileEncoder, stdoutEncoder, cfg.Output)
	appCore := buildCore(cfg.App, level, fileEncoder, stdoutEncoder, cfg.Output)
	errorCore := buildCore(cfg.Error, zap.NewAtomicLevelAt(zap.ErrorLevel), fileEncoder, stdoutEncoder, cfg.Output)

	// Compose: app core for all logs, error core for error+ only
	mainCore := zapcore.NewTee(appCore, errorCore)

	accessLogger := zap.New(accessCore, zap.AddCaller(), zap.AddCallerSkip(1))
	appLogger := zap.New(mainCore, zap.AddCaller(), zap.AddCallerSkip(1))
	errorLogger := zap.New(errorCore, zap.AddCaller(), zap.AddCallerSkip(1))

	// Set global logger
	zap.ReplaceGlobals(appLogger)

	return &Logger{
		Access: accessLogger,
		App:    appLogger,
		Error:  errorLogger,
		level:  level,
	}
}

// buildCore creates a zapcore.Core with separate file and stdout writers.
// In "both" mode, file gets clean output (no ANSI colors) while stdout gets colored output.
func buildCore(fileCfg config.LogFileConfig, level zap.AtomicLevel, fileEncoder, stdoutEncoder zapcore.Encoder, output string) zapcore.Core {
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
// Returns a no-op WriteSyncer if file logging is disabled to prevent silent output to stdout.
func newFileWriter(cfg config.LogFileConfig) zapcore.WriteSyncer {
	if !cfg.Enabled || cfg.Path == "" {
		return zapcore.AddSync(io.Discard)
	}

	// Time-based (daily) rotation with date-named files
	if cfg.TimeBased {
		return zapcore.AddSync(newDailyRotateSyncer(cfg))
	}

	// Size-based rotation via lumberjack (original behavior)
	dir := filepath.Dir(cfg.Path)
	if dir != "" {
		_ = os.MkdirAll(dir, 0o755)
	}
	lj := &lumberjack.Logger{
		Filename:   cfg.Path,
		MaxSize:    cfg.MaxSize,
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAge,
		Compress:   cfg.Compress,
	}
	return zapcore.AddSync(lj)
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
