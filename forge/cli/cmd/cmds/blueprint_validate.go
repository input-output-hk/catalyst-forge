package cmds

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

type ValidateCmd struct {
	Config string `arg:"" help:"Path to the blueprint file."`
}

func (c *ValidateCmd) Run(logger *slog.Logger) error {
	if _, err := os.Stat(c.Config); os.IsNotExist(err) {
		return fmt.Errorf("blueprint file does not exist: %s", c.Config)
	}

	_, err := loadBlueprint(filepath.Dir(c.Config), logger)
	if err != nil {
		return err
	}

	return nil
}
