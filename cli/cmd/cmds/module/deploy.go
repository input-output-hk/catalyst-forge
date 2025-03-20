package module

import (
	"fmt"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/events"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/lib/project/deployment"
	"github.com/input-output-hk/catalyst-forge/lib/project/deployment/deployer"
	"github.com/input-output-hk/catalyst-forge/lib/tools/fs"
)

type DeployCmd struct {
	Force   bool   `help:"Force deployment even if no deployment event is firing."`
	Project string `arg:"" help:"The path to the project to deploy." kong:"arg,predictor=path"`
}

func (c *DeployCmd) Run(ctx run.RunContext) error {
	exists, err := fs.Exists(c.Project)
	if err != nil {
		return fmt.Errorf("could not check if project exists: %w", err)
	} else if !exists {
		return fmt.Errorf("project does not exist: %s", c.Project)
	}

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

	d := deployer.NewDeployer(
		deployer.NewDeployerConfigFromProject(&project),
		ctx.ManifestGeneratorStore,
		ctx.SecretStore,
		ctx.Logger,
		ctx.CueCtx,
	)

	dr, err := d.CreateDeployment(project.Name, project.Name, deployment.NewModuleBundle(&project))
	if err != nil {
		return fmt.Errorf("failed creating deployment: %w", err)
	}

	if !dryrun {
		changes, err := dr.HasChanges()
		if err != nil {
			return fmt.Errorf("failed checking for changes: %w", err)
		}

		if !changes {
			ctx.Logger.Warn("no changes to deploy")
			return nil
		}

		if err := dr.Commit(); err != nil {
			return fmt.Errorf("failed committing deployment: %w", err)
		}
	} else {
		ctx.Logger.Info("Dry-run: not committing or pushing changes")
		ctx.Logger.Info("Dumping manifests")
		for _, r := range dr.Manifests {
			fmt.Println(string(r))
		}
	}

	return nil
}
