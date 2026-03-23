package github

import (
	"context"
	"fmt"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/client"
)

type DeleteCmd struct {
	ID uint `arg:"" help:"The ID of the authentication entry to delete."`
}

func (c *DeleteCmd) Run(ctx run.RunContext, cl client.Client) error {
	err := cl.Github().DeleteAuth(context.Background(), c.ID)
	if err != nil {
		return fmt.Errorf("failed to delete authentication entry: %w", err)
	}

	ctx.Logger.Info("Authentication entry deleted", "id", c.ID)
	return nil
}
