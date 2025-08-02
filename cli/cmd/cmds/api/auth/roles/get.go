package roles

import (
	"context"
	"fmt"
	"strconv"

	"github.com/input-output-hk/catalyst-forge/cli/cmd/cmds/api/auth/common"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/foundry/api/client"
)

type GetCmd struct {
	ID   *string `short:"i" help:"The ID of the role to retrieve."`
	Name *string `short:"n" help:"The name of the role to retrieve."`
	JSON bool    `short:"j" help:"Output as prettified JSON instead of table."`
}

func (c *GetCmd) Run(ctx run.RunContext, cl client.Client) error {
	if c.ID == nil && c.Name == nil {
		return fmt.Errorf("either --id or --name must be specified")
	}

	if c.ID != nil && c.Name != nil {
		return fmt.Errorf("only one of --id or --name can be specified")
	}

	role, err := c.retrieveRole(cl)
	if err != nil {
		return err
	}

	if c.JSON {
		return common.OutputRoleJSON(role)
	}

	return common.OutputRoleTable(role)
}

func (c *GetCmd) retrieveRole(cl client.Client) (*client.Role, error) {
	if c.ID != nil {
		// Convert string ID to uint
		id, err := strconv.ParseUint(*c.ID, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid ID format: %w", err)
		}
		role, err := cl.GetRole(context.Background(), uint(id))
		if err != nil {
			return nil, fmt.Errorf("failed to get role by ID: %w", err)
		}
		return role, nil
	}

	role, err := cl.GetRoleByName(context.Background(), *c.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to get role by name: %w", err)
	}
	return role, nil
}
