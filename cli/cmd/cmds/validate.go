package cmds

import (
	"log/slog"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
)

type ValidateCmd struct {
	Project string `arg:"" help:"Path to the project."`
}

func (c *ValidateCmd) Run(ctx run.RunContext, logger *slog.Logger) error {
	_, err := loadProject(ctx, c.Project, logger)
	if err != nil {
		return err
	}

	return nil
}
