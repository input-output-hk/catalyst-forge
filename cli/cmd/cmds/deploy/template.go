package deploy

import (
	"fmt"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/deployment"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
)

type TemplateCmd struct {
	Project string `arg:"" help:"The path to the project." kong:"arg,predictor=path"`
	Values  bool   `help:"Only print the values.yml for the main module"`
}

func (c *TemplateCmd) Run(ctx run.RunContext) error {
	project, err := ctx.ProjectLoader.Load(c.Project)
	if err != nil {
		return fmt.Errorf("could not load project: %w", err)
	}

	runner := deployment.NewKCLRunner(ctx.Logger)

	if c.Values {
		values, err := runner.GetMainValues(&project)
		if err != nil {
			return fmt.Errorf("could not get values: %w", err)
		}

		fmt.Print(values)
		return nil
	}

	out, err := runner.RunDeployment(&project)
	if err != nil {
		return fmt.Errorf("could not run deployment: %w", err)
	}

	fmt.Print(out)
	return nil
}
