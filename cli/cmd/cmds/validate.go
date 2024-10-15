package cmds

import (
	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
)

type ValidateCmd struct {
	Project string `arg:"" help:"Path to the project."`
}

func (c *ValidateCmd) Run(ctx run.RunContext) error {
	_, err := ctx.ProjectLoader.Load(c.Project)
	if err != nil {
		return err
	}

	return nil
}
