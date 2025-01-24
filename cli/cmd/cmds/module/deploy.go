package module

import (
	"fmt"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/deployment"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/events"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
)

type DeployCmd struct {
	Force   bool   `help:"Force deployment even if no deployment event is firing."`
	Project string `arg:"" help:"The path to the project to deploy." kong:"arg,predictor=path"`
}

func (c *DeployCmd) Run(ctx run.RunContext) error {
	project, err := ctx.ProjectLoader.Load(c.Project)
	if err != nil {
		return fmt.Errorf("could not load project: %w", err)
	}

	var dryrun bool
	eh := events.NewDefaultEventHandler(ctx.Logger)
	if !eh.Firing(&project, project.GetDeploymentEvents()) && !c.Force {
		ctx.Logger.Info("No deployment event is firing, performing dry-run")
		dryrun = true
	}

	deployer := deployment.NewGitopsDeployer(&project, &ctx.SecretStore, ctx.DeploymentGenerator, ctx.Logger, dryrun)
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
