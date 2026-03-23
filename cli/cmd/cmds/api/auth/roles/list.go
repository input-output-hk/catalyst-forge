package roles

import (
	"context"
	"fmt"

	"github.com/input-output-hk/catalyst-forge/cli/cmd/cmds/api/auth/common"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/client"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/client/users"
)

type ListCmd struct {
	JSON bool `short:"j" help:"Output as prettified JSON instead of table."`
}

func (c *ListCmd) Run(ctx run.RunContext, cl client.Client) error {
	roles, err := c.listRoles(cl)
	if err != nil {
		return err
	}

	if c.JSON {
		return common.OutputRolesJSON(roles)
	}

	return common.OutputRolesTable(roles)
}

// listRoles retrieves all roles.
func (c *ListCmd) listRoles(cl client.Client) ([]users.Role, error) {
	roles, err := cl.Roles().List(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to list roles: %w", err)
	}

	return roles, nil
}
