package cmds

import (
	"fmt"
	"log/slog"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/deployment"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/secrets"
)

type DeployCmd struct {
	Project string `arg:"" help:"The path to the project to deploy."`
}

func (c *DeployCmd) Run(logger *slog.Logger, global GlobalArgs) error {
	project, err := loadProject(global, c.Project, logger)
	if err != nil {
		return fmt.Errorf("could not load project: %w", err)
	}

	store := secrets.NewDefaultSecretStore()
	deployer := deployment.NewGitopsDeployer(&project, &store, logger)
	if err := deployer.Load(); err != nil {
		return fmt.Errorf("could not load deployer: %w", err)
	}

	if err := deployer.Deploy(); err != nil {
		if err == deployment.ErrNoChanges {
			logger.Warn("no changes to deploy")
			return nil
		}

		return fmt.Errorf("could not deploy project: %w", err)
	}

	return nil
}
