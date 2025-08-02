package gha

import (
	"context"
	"fmt"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/foundry/api/client"
	"github.com/input-output-hk/catalyst-forge/foundry/api/client/gha"
	"github.com/input-output-hk/catalyst-forge/foundry/api/pkg/auth"
)

type UpdateCmd struct {
	ID          uint              `arg:"" help:"The ID of the authentication entry to update."`
	Admin       *bool             `short:"a" help:"Whether the authentication entry is an admin entry."`
	Enabled     *bool             `short:"e" help:"Whether the authentication entry is enabled."`
	Description *string           `short:"d" help:"The description of the authentication entry."`
	Permissions []auth.Permission `short:"p" help:"The permissions to grant to the authentication entry."`
	JSON        bool              `short:"j" help:"Output as prettified JSON instead of table."`
}

func (c *UpdateCmd) Run(ctx run.RunContext, cl client.Client) error {
	// Build the update request
	req := &gha.UpdateAuthRequest{}

	if c.Admin != nil {
		if *c.Admin {
			req.Permissions = auth.AllPermissions
		} else {
			req.Permissions = c.Permissions
		}
	} else if len(c.Permissions) > 0 {
		req.Permissions = c.Permissions
	}

	if c.Enabled != nil {
		req.Enabled = *c.Enabled
	}

	if c.Description != nil {
		req.Description = *c.Description
	}

	auth, err := cl.GHA().UpdateAuth(context.Background(), c.ID, req)
	if err != nil {
		return fmt.Errorf("failed to update authentication entry: %w", err)
	}

	ctx.Logger.Info("Authentication entry updated", "id", auth.ID)

	if c.JSON {
		return outputJSON(auth)
	}

	return outputTable(auth)
}
