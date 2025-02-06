package module

import (
	"fmt"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/events"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/lib/project/deployment/deployer"
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

	d := deployer.NewDeployer(&project, ctx.ManifestGeneratorStore, ctx.Logger, dryrun)
	if err := d.Deploy(); err != nil {
		if err == deployer.ErrNoChanges {
			ctx.Logger.Warn("no changes to deploy")
			return nil
		}

		return fmt.Errorf("failed deploying project: %w", err)
	}

	return nil
}
