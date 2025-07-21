package release

import (
	"context"
	"fmt"
	"time"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/utils"
	api "github.com/input-output-hk/catalyst-forge/foundry/api/client"
)

type ReleaseGetCmd struct {
	ReleaseID string `arg:"" help:"The ID of the release."`
	Url       string `short:"u" help:"The URL to the Foundry API server (overrides global config)."`
}

func (c *ReleaseGetCmd) Run(ctx run.RunContext) error {
	url, err := utils.GetFoundryURL(ctx, c.Url)
	if err != nil {
		return err
	}

	client := api.NewClient(url, api.WithTimeout(10*time.Second))
	release, err := client.GetRelease(context.Background(), c.ReleaseID)
	if err != nil {
		return fmt.Errorf("could not show release: %w", err)
	}

	if err := utils.PrintJson(release, true); err != nil {
		return err
	}

	return nil
}
