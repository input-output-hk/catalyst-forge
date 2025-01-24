package module

import (
	"fmt"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/deployment"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
)

type ValuesCmd struct {
	Module string `arg:"" help:"The path to the module (or project)." kong:"arg,predictor=path"`
	Name   string `short:"n" help:"The name of the module to get values for." default:"main"`
}

func (c *ValuesCmd) Run(ctx run.RunContext) error {
	project, err := ctx.ProjectLoader.Load(c.Module)
	if err != nil {
		return fmt.Errorf("could not load project: %w", err)
	}

	runner := deployment.NewKCLRunner(ctx.Logger)
	values, err := runner.GetMainValues(&project, c.Name)
	if err != nil {
		return fmt.Errorf("could not get values: %w", err)
	}

	fmt.Print(values)

	return nil
}
