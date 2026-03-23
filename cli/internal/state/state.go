package state

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
)

// GetDir gets the state directory for the CLI.
// It creates the directory if it doesn't exist.
func GetDir(ctx run.RunContext) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	path := filepath.Join(home, ".local", "state", "forge")
	exists, err := ctx.FS.Exists(path)
	if err != nil {
		return "", fmt.Errorf("failed to check if state directory exists: %w", err)
	} else if !exists {
		ctx.Logger.Info("Creating state directory", "path", path)
		if err := ctx.FS.MkdirAll(path, 0755); err != nil {
			return "", fmt.Errorf("failed to create state directory: %s", err)
		}
	}

	return path, nil
}
