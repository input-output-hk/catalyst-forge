package roles

import (
	"context"
	"fmt"

	"github.com/input-output-hk/catalyst-forge/cli/cmd/cmds/api/auth/common"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/foundry/api/client"
)

type CreateCmd struct {
	Name        string   `arg:"" help:"The name of the role to create."`
	Permissions []string `short:"p" help:"The permissions to grant to the role."`
	Admin       bool     `short:"a" help:"Create role with admin privileges (all permissions)."`
	JSON        bool     `short:"j" help:"Output as prettified JSON instead of table."`
}

func (c *CreateCmd) Run(ctx run.RunContext, cl client.Client) error {
	var role *client.Role
	var err error

	if c.Admin {
		role, err = cl.CreateRoleWithAdmin(context.Background(), &client.CreateRoleRequest{
			Name:        c.Name,
			Permissions: c.Permissions, // This will be ignored when admin=true
		})
	} else {
		role, err = cl.CreateRole(context.Background(), &client.CreateRoleRequest{
			Name:        c.Name,
			Permissions: c.Permissions,
		})
	}

	if err != nil {
		return fmt.Errorf("failed to create role: %w", err)
	}

	if c.JSON {
		return common.OutputRoleJSON(role)
	}

	return common.OutputRoleTable(role)
}
