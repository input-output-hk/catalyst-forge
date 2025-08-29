package logging

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

// Options controls logger construction.
type Options struct {
	Level     string
	Format    string // "json" or "text"
	AddSource bool
	Writer    io.Writer // optional; defaults to os.Stdout
}

var root *slog.Logger

// Setup initializes the global logger according to options.
func Setup(opts Options) *slog.Logger {
	var lvl slog.Level
	switch strings.ToLower(opts.Level) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	w := opts.Writer
	if w == nil {
		w = os.Stdout
	}

	handlerOpts := &slog.HandlerOptions{
		Level:     lvl,
		AddSource: opts.AddSource,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			k := strings.ToLower(a.Key)
			if k == "raw_query" || k == "authorization" || strings.Contains(k, "token") || strings.Contains(k, "consent_challenge") || strings.Contains(k, "login_challenge") {
				return slog.Attr{Key: a.Key, Value: slog.StringValue("[redacted]")}
			}
			return a
		},
	}
	var handler slog.Handler
	switch strings.ToLower(opts.Format) {
	case "json":
		handler = slog.NewJSONHandler(w, handlerOpts)
	default:
		handler = slog.NewTextHandler(w, handlerOpts)
	}

	root = slog.New(handler)
	return root
}

// L returns the process-wide logger. Setup must be called first.
func L() *slog.Logger {
	if root == nil {
		root = slog.Default()
	}
	return root
}
