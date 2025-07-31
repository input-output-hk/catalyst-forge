package gha

import (
	"context"
	"fmt"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/foundry/api/client"
	"github.com/input-output-hk/catalyst-forge/foundry/api/pkg/auth"
)

type CreateCmd struct {
	Admin       bool              `short:"a" help:"Whether the authentication entry is an admin entry." default:"false"`
	Enabled     bool              `short:"e" help:"Whether the authentication entry is enabled." default:"true"`
	Description string            `short:"d" help:"The description of the authentication entry." default:""`
	Repository  string            `arg:"" help:"The repository to create the authentication entry for."`
	Permissions []auth.Permission `short:"p" help:"The permissions to grant to the authentication entry."`
	JSON        bool              `short:"j" help:"Output as prettified JSON instead of table."`
}

func (c *CreateCmd) Run(ctx run.RunContext, cl client.Client) error {
	var permissions []auth.Permission
	if c.Admin {
		permissions = auth.AllPermissions
	} else {
		permissions = c.Permissions
	}

	auth, err := cl.CreateAuth(context.Background(), &client.CreateAuthRequest{
		Repository:  c.Repository,
		Permissions: permissions,
		Description: c.Description,
		Enabled:     c.Enabled,
	})
	if err != nil {
		return fmt.Errorf("failed to create authentication entry: %w", err)
	}

	if c.JSON {
		return outputJSON(auth)
	}

	return outputTable(auth)
}
