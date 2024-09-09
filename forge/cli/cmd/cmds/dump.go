package cmds

import (
	"fmt"
	"log/slog"
)

type DumpCmd struct {
	Project string `arg:"" help:"Path to the project."`
	Pretty  bool   `help:"Pretty print JSON output."`
}

func (c *DumpCmd) Run(logger *slog.Logger) error {
	project, err := loadProject(c.Project, logger)
	if err != nil {
		return fmt.Errorf("could not load project: %w", err)
	}

	json, err := project.Raw().MarshalJSON()
	if err != nil {
		return err
	}

	fmt.Println(string(json))
	return nil
}
