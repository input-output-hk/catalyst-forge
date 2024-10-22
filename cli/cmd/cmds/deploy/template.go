package deploy

import (
	"fmt"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
)

type TemplateCmd struct {
	Project string `arg:"" help:"The path to the project." kong:"arg,predictor=path"`
}

func (c *TemplateCmd) Run(ctx run.RunContext) error {
	_, err := ctx.ProjectLoader.Load(c.Project)
	if err != nil {
		return fmt.Errorf("could not load project: %w", err)
	}

	return nil
}
