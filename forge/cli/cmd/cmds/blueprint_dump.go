package cmds

import (
	"fmt"
	"log/slog"
	"os"
)

type DumpCmd struct {
	Blueprint string `arg:"" help:"Path to the blueprint file."`
	Pretty    bool   `help:"Pretty print JSON output."`
}

func (c *DumpCmd) Run(logger *slog.Logger) error {
	if _, err := os.Stat(c.Blueprint); os.IsNotExist(err) {
		return fmt.Errorf("blueprint file does not exist: %s", c.Blueprint)
	}

	rbp, err := loadRawBlueprint(c.Blueprint, logger)
	if err != nil {
		return fmt.Errorf("could not load blueprint: %w", err)
	}

	json, err := rbp.MarshalJSON()
	if err != nil {
		return err
	}

	fmt.Println(string(json))
	return nil
}
