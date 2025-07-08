package release

import (
	"context"
	"fmt"
	"time"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/utils"
	api "github.com/input-output-hk/catalyst-forge/foundry/api/client"
)

type ReleaseListCmd struct {
	Project string `arg:"" help:"The project to list releases for."`
	Url     string `short:"u" help:"The URL to the Foundry API server (overrides global config)."`
}

func (c *ReleaseListCmd) Run(ctx run.RunContext) error {
	url, err := utils.GetFoundryURL(ctx, c.Url)
	if err != nil {
		return err
	}

	client := api.NewClient(url, api.WithTimeout(10*time.Second))
	releases, err := client.ListReleases(context.Background(), c.Project)
	if err != nil {
		return fmt.Errorf("could not list releases: %w", err)
	}

	if err := utils.PrintJson(releases, true); err != nil {
		return err
	}

	return nil
}
