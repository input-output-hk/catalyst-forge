package module

import (
	"fmt"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/events"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/lib/deployment"
	"github.com/input-output-hk/catalyst-forge/lib/deployment/deployer"
	"github.com/input-output-hk/catalyst-forge/lib/tools/fs"
	"github.com/spf13/cobra"
)

// DeployOptions holds the flags for the module deploy command.
type DeployOptions struct {
	Force bool
}

// NewDeployCommand creates the module deploy subcommand.
func NewDeployCommand() *cobra.Command {
	opts := &DeployOptions{}

	cmd := &cobra.Command{
		Use:   "deploy PROJECT",
		Short: "Deploys a project to the configured GitOps repository",
		Long:  `Deploy project modules to GitOps repository with automatic change detection and dry-run capabilities.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := run.MustFromContext(cmd.Context())
			return deployExecute(ctx, args[0], opts)
		},
	}

	cmd.Flags().BoolVar(&opts.Force, "force", false, "Force deployment even if no deployment event is firing")

	return cmd
}

// deployExecute executes the module deploy command logic.
func deployExecute(ctx run.RunContext, projectPath string, opts *DeployOptions) error {
	exists, err := fs.Exists(projectPath)
	if err != nil {
		return fmt.Errorf("could not check if project exists: %w", err)
	} else if !exists {
		return fmt.Errorf("project does not exist: %s", projectPath)
	}

	project, err := ctx.ProjectLoader.Load(projectPath)
	if err != nil {
		return fmt.Errorf("could not load project: %w", err)
	}

	var dryrun bool
	eh := events.NewDefaultEventHandler(ctx.Logger)
	if !eh.Firing(&project, project.GetDeploymentEvents()) && !opts.Force {
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
