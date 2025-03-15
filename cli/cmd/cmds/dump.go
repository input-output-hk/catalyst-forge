package cmds

import (
	"fmt"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/lib/tools/fs"
)

type DumpCmd struct {
	Project string `arg:"" help:"Path to the project." kong:"arg,predictor=path"`
	Pretty  bool   `help:"Pretty print JSON output."`
}

func (c *DumpCmd) Run(ctx run.RunContext) error {
	exists, err := fs.Exists(c.Project)
	if err != nil {
		return fmt.Errorf("could not check if project exists: %w", err)
	} else if !exists {
		return fmt.Errorf("project does not exist: %s", c.Project)
	}

	project, err := ctx.ProjectLoader.Load(c.Project)
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
