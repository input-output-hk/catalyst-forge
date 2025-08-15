package logging

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"
)

// Logger is the interface for structured logging in the KCL library.
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	With(args ...any) Logger
	WithContext(ctx context.Context) Logger
}

// logger wraps slog.Logger to provide our interface.
type logger struct {
	sl *slog.Logger
}

// Default logger instance
var defaultLogger Logger

func init() {
	// Initialize with JSON handler by default
	opts := &slog.HandlerOptions{
		Level: getLogLevel(),
	}
	
	var handler slog.Handler
	if os.Getenv("FORGE_KCL_LOG_FORMAT") == "text" {
		handler = slog.NewTextHandler(os.Stderr, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stderr, opts)
	}
	
	defaultLogger = &logger{
		sl: slog.New(handler),
	}
}

// getLogLevel returns the log level from environment.
func getLogLevel() slog.Level {
	switch os.Getenv("FORGE_KCL_LOG_LEVEL") {
	case "debug", "DEBUG":
		return slog.LevelDebug
	case "info", "INFO":
		return slog.LevelInfo
	case "warn", "WARN":
		return slog.LevelWarn
	case "error", "ERROR":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// GetLogger returns the default logger.
func GetLogger() Logger {
	return defaultLogger
}

// SetLogger sets the default logger.
func SetLogger(l Logger) {
	defaultLogger = l
}

// Debug logs at debug level.
func (l *logger) Debug(msg string, args ...any) {
	l.sl.Debug(msg, args...)
}

// Info logs at info level.
func (l *logger) Info(msg string, args ...any) {
	l.sl.Info(msg, args...)
}

// Warn logs at warning level.
func (l *logger) Warn(msg string, args ...any) {
	l.sl.Warn(msg, args...)
}

// Error logs at error level.
func (l *logger) Error(msg string, args ...any) {
	l.sl.Error(msg, args...)
}

// With returns a logger with additional fields.
func (l *logger) With(args ...any) Logger {
	return &logger{
		sl: l.sl.With(args...),
	}
}

// WithContext returns a logger with context fields.
func (l *logger) WithContext(ctx context.Context) Logger {
	// Extract trace ID if available
	if traceID := ctx.Value("trace_id"); traceID != nil {
		return l.With("trace_id", traceID)
	}
	return l
}

// Helper functions for common logging patterns

// LogOperation logs the start and end of an operation.
func LogOperation(ctx context.Context, operation string, fn func() error) error {
	log := GetLogger().WithContext(ctx)
	log.Debug(fmt.Sprintf("Starting %s", operation))
	
	err := fn()
	
	if err != nil {
		log.Error(fmt.Sprintf("Failed %s", operation), "error", err)
	} else {
		log.Debug(fmt.Sprintf("Completed %s", operation))
	}
	
	return err
}

// LogDuration logs the duration of an operation.
func LogDuration(log Logger, operation string, start int64) {
	duration := nowUnixMilli() - start
	log.Debug(fmt.Sprintf("Operation %s completed", operation), 
		"duration_ms", duration)
}

// nowUnixMilli returns current time in milliseconds.
func nowUnixMilli() int64 {
	return time.Now().UnixMilli()
}