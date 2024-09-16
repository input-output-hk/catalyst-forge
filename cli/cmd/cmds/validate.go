package cmds

import (
	"log/slog"
)

type ValidateCmd struct {
	Project string `arg:"" help:"Path to the project."`
}

func (c *ValidateCmd) Run(logger *slog.Logger, global GlobalArgs) error {
	_, err := loadProject(global, c.Project, logger)
	if err != nil {
		return err
	}

	return nil
}
