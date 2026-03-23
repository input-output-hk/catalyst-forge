package roles

import (
	"context"
	"fmt"

	"github.com/input-output-hk/catalyst-forge/cli/cmd/cmds/api/auth/common"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/client"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/client/users"
)

type CreateCmd struct {
	Name        string   `short:"n" help:"The name of the role to create." required:"true"`
	Permissions []string `short:"p" help:"The permissions to grant to the role (mutually exclusive with --admin)."`
	Admin       bool     `short:"a" help:"Create role with admin privileges (all permissions) (mutually exclusive with --permissions)."`
	JSON        bool     `short:"j" help:"Output as prettified JSON instead of table."`
}

func (c *CreateCmd) Run(ctx run.RunContext, cl client.Client) error {
	if c.Admin && len(c.Permissions) > 0 {
		return fmt.Errorf("only one of --admin or --permissions can be specified")
	}

	role, err := c.createRole(cl)
	if err != nil {
		return err
	}

	if c.JSON {
		return common.OutputRoleJSON(role)
	}

	return common.OutputRoleTable(role)
}

// createRole creates a new role with the specified parameters.
func (c *CreateCmd) createRole(cl client.Client) (*users.Role, error) {
	var role *users.Role
	var err error

	if c.Admin {
		role, err = cl.Roles().CreateWithAdmin(context.Background(), &users.CreateRoleRequest{
			Name:        c.Name,
			Permissions: c.Permissions,
		})
	} else {
		role, err = cl.Roles().Create(context.Background(), &users.CreateRoleRequest{
			Name:        c.Name,
			Permissions: c.Permissions,
		})
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create role: %w", err)
	}

	return role, nil
}
