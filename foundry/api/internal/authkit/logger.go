package authkit

import (
	"context"
	"log/slog"
)

// slogLogger adapts slog.Logger to AuthKit's Logger interface.
type slogLogger struct{ l *slog.Logger }

func NewLogger(l *slog.Logger) *slogLogger { return &slogLogger{l: l} }

func (s *slogLogger) Debug(ctx context.Context, msg string, fields ...any) {
	s.l.Log(ctx, slog.LevelDebug, msg, fields...)
}
func (s *slogLogger) Info(ctx context.Context, msg string, fields ...any) {
	s.l.Log(ctx, slog.LevelInfo, msg, fields...)
}
func (s *slogLogger) Warn(ctx context.Context, msg string, fields ...any) {
	s.l.Log(ctx, slog.LevelWarn, msg, fields...)
}
func (s *slogLogger) Error(ctx context.Context, msg string, fields ...any) {
	s.l.Log(ctx, slog.LevelError, msg, fields...)
}
