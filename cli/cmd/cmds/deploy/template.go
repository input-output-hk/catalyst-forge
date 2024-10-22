package deploy

import (
	"fmt"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/deployment"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
)

type TemplateCmd struct {
	Project string `arg:"" help:"The path to the project." kong:"arg,predictor=path"`
}

func (c *TemplateCmd) Run(ctx run.RunContext) error {
	project, err := ctx.ProjectLoader.Load(c.Project)
	if err != nil {
		return fmt.Errorf("could not load project: %w", err)
	}

	bundle, err := deployment.GenerateBundle(&project)
	if err != nil {
		return fmt.Errorf("could not generate bundle: %w", err)
	}

	templater, err := deployment.NewDefaultBundleTemplater(ctx.Logger)
	if err != nil {
		return fmt.Errorf("could not create bundle templater: %w", err)
	}

	out, err := templater.Render(bundle)
	if err != nil {
		return fmt.Errorf("could not render bundle: %w", err)
	}

	fmt.Println(out)

	return nil
}
