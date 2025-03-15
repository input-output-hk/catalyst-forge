package cmds

import (
	"fmt"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/lib/tools/fs"
)

type ValidateCmd struct {
	Project string `kong:"arg,predictor=path" help:"Path to the project."`
}

func (c *ValidateCmd) Run(ctx run.RunContext) error {
	exists, err := fs.Exists(c.Project)
	if err != nil {
		return fmt.Errorf("could not check if project exists: %w", err)
	} else if !exists {
		return fmt.Errorf("project does not exist: %s", c.Project)
	}

	_, err = ctx.ProjectLoader.Load(c.Project)
	if err != nil {
		return err
	}

	return nil
}
