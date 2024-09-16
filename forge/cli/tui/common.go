package tui

import (
	"log/slog"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Window represents the dimensions of a terminal window.
type Window struct {
	Height int
	Width  int
}

// Resize updates the dimensions of the window.
func (w *Window) Resize(msg tea.WindowSizeMsg) {
	w.Height, w.Width = msg.Height, msg.Width
}

// NewLogger creates a new logger and returns it along with the file it writes
// to. The file should be closed when the logger is no longer needed.
func NewLogger() (*slog.Logger, *os.File, error) {
	f, err := os.Create("debug.log")
	if err != nil {
		return nil, nil, err
	}

	options := slog.HandlerOptions{
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				// Format time as HH:MM:SS instead of the default RFC3339 format
				a.Value = slog.StringValue(time.Now().Format("15:04:05"))
			}
			return a
		},
	}

	return slog.New(slog.NewTextHandler(f, &options)), f, nil
}
