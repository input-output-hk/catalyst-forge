package cmds

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

type DumpCmd struct {
	Config string `arg:"" help:"Path to the configuration file."`
	Pretty bool   `help:"Pretty print JSON output."`
}

func (c *DumpCmd) Run(logger *slog.Logger) error {
	if _, err := os.Stat(c.Config); os.IsNotExist(err) {
		return fmt.Errorf("configuration file does not exist: %s", c.Config)
	}

	config, err := loadBlueprint(filepath.Dir(c.Config), logger)
	if err != nil {
		return err
	}

	printJson(config, c.Pretty)
	return nil
}
