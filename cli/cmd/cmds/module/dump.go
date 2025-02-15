package module

import (
	"fmt"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/lib/project/deployment"
)

type DumpCmd struct {
	Project string `arg:"" help:"The path to the project to dump." kong:"arg,predictor=path"`
}

func (c *DumpCmd) Run(ctx run.RunContext) error {
	project, err := ctx.ProjectLoader.Load(c.Project)
	if err != nil {
		return fmt.Errorf("could not load project: %w", err)
	}

	bundle := deployment.NewDeploymentBundle(project)
	result, err := bundle.Dump()
	if err != nil {
		return fmt.Errorf("failed to dump deployment modules: %w", err)
	}

	// modules := project.Blueprint.Project.Deployment.Modules
	// if modules == nil {
	// 	return fmt.Errorf("no deployment modules found for project")
	// }

	// result, err := deployment.DumpBundle(modules)
	// if err != nil {
	// 	return fmt.Errorf("failed to dump deployment modules: %w", err)
	// }

	fmt.Print(string(result))
	return nil
}
