package deploy

import (
	"context"
	"fmt"
	"time"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/utils"
	api "github.com/input-output-hk/catalyst-forge/foundry/api/client"
)

type DeployCreateCmd struct {
	ReleaseID string `arg:"" help:"The release ID to deploy."`
	Url       string `short:"u" help:"The URL to the Foundry API server (overrides global config)."`
}

func (c *DeployCreateCmd) Run(ctx run.RunContext) error {
	url, err := utils.GetFoundryURL(ctx, c.Url)
	if err != nil {
		return err
	}

	client := api.NewClient(url, api.WithTimeout(10*time.Second))
	deployment, err := client.CreateDeployment(context.Background(), c.ReleaseID)
	if err != nil {
		return fmt.Errorf("could not create deployment: %w", err)
	}

	if err := utils.PrintJson(deployment, true); err != nil {
		return err
	}

	return nil
}
