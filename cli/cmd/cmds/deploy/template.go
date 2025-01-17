package deploy

import (
	"fmt"
	"strings"

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

	result, err := runner.RunDeployment(&project)
	if err != nil {
		return fmt.Errorf("could not run deployment: %w", err)
	}

	var out string
	for _, module := range result {
		out += fmt.Sprintf("%s---\n", module.Manifests)
	}

	fmt.Print(strings.TrimSuffix(out, "---\n"))

	return nil
}
