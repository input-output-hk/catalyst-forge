package module

import (
	"fmt"
	"strings"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/deployment"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
)

type TemplateCmd struct {
	Module string `arg:"" help:"The path to the module (or project)." kong:"arg,predictor=path"`
}

func (c *TemplateCmd) Run(ctx run.RunContext) error {
	project, err := ctx.ProjectLoader.Load(c.Module)
	if err != nil {
		return fmt.Errorf("could not load project: %w", err)
	}

	runner := deployment.NewKCLRunner(ctx.Logger)
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
