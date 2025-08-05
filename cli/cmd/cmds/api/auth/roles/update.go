package roles

import (
	"context"
	"fmt"
	"strconv"

	"github.com/input-output-hk/catalyst-forge/cli/cmd/cmds/api/auth/common"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/client"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/client/users"
)

type UpdateCmd struct {
	ID          *string  `short:"i" help:"The numeric ID of the role to update (mutually exclusive with --name)."`
	Name        *string  `short:"n" help:"The name of the role to update (mutually exclusive with --id)."`
	NewName     *string  `short:"r" help:"The new name for the role."`
	Permissions []string `short:"p" help:"The new permissions for the role."`
	JSON        bool     `short:"j" help:"Output as prettified JSON instead of table."`
}

func (c *UpdateCmd) Run(ctx run.RunContext, cl client.Client) error {
	if c.ID == nil && c.Name == nil {
		return fmt.Errorf("either --id or --name must be specified")
	}

	if c.ID != nil && c.Name != nil {
		return fmt.Errorf("only one of --id or --name can be specified")
	}

	role, err := c.updateRole(cl)
	if err != nil {
		return err
	}

	if c.JSON {
		return common.OutputRoleJSON(role)
	}

	return common.OutputRoleTable(role)
}

// updateRole updates a role by ID or name.
func (c *UpdateCmd) updateRole(cl client.Client) (*users.Role, error) {
	var roleID uint

	if c.Name != nil {
		role, err := cl.Roles().GetByName(context.Background(), *c.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to get role by name: %w", err)
		}
		roleID = role.ID
	} else if c.ID != nil {
		parsedID, err := strconv.ParseUint(*c.ID, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid ID format: %w", err)
		}
		roleID = uint(parsedID)
	}

	req := &users.UpdateRoleRequest{}

	var currentRole *users.Role
	var err error
	if c.Name != nil {
		currentRole, err = cl.Roles().GetByName(context.Background(), *c.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to get role by name: %w", err)
		}
	} else if c.ID != nil {
		currentRole, err = cl.Roles().Get(context.Background(), roleID)
		if err != nil {
			return nil, fmt.Errorf("failed to get role by ID: %w", err)
		}
	}

	if c.NewName != nil {
		req.Name = *c.NewName
	} else {
		req.Name = currentRole.Name
	}

	if len(c.Permissions) > 0 {
		req.Permissions = c.Permissions
	} else {
		// If no permissions provided, use the current permissions
		req.Permissions = currentRole.Permissions
	}

	role, err := cl.Roles().Update(context.Background(), roleID, req)
	if err != nil {
		return nil, fmt.Errorf("failed to update role: %w", err)
	}

	return role, nil
}
