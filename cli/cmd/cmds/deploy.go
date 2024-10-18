package cmds

import (
	"fmt"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/deployment"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
)

type DeployCmd struct {
	Project string `arg:"" help:"The path to the project to deploy." kong:"arg,predictor=path"`
}

func (c *DeployCmd) Run(ctx run.RunContext) error {
	project, err := ctx.ProjectLoader.Load(c.Project)
	if err != nil {
		return fmt.Errorf("could not load project: %w", err)
	}

	deployer := deployment.NewGitopsDeployer(&project, &ctx.SecretStore, ctx.Logger)
	if err := deployer.Load(); err != nil {
		return fmt.Errorf("could not load deployer: %w", err)
	}

	if err := deployer.Deploy(); err != nil {
		if err == deployment.ErrNoChanges {
			ctx.Logger.Warn("no changes to deploy")
			return nil
		}

		return fmt.Errorf("could not deploy project: %w", err)
	}

	return nil
}
