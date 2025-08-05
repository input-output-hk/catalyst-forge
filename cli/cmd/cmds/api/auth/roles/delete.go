package roles

import (
	"context"
	"fmt"
	"strconv"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/client"
)

type DeleteCmd struct {
	ID   *string `short:"i" help:"The numeric ID of the role to delete (mutually exclusive with --name)."`
	Name *string `short:"n" help:"The name of the role to delete (mutually exclusive with --id)."`
}

func (c *DeleteCmd) Run(ctx run.RunContext, cl client.Client) error {
	if c.ID == nil && c.Name == nil {
		return fmt.Errorf("either --id or --name must be specified")
	}

	if c.ID != nil && c.Name != nil {
		return fmt.Errorf("only one of --id or --name can be specified")
	}

	err := c.deleteRole(cl)
	if err != nil {
		return err
	}

	identifier := ""
	if c.ID != nil {
		identifier = *c.ID
	} else {
		identifier = *c.Name
	}

	fmt.Printf("Role %s deleted successfully.\n", identifier)
	return nil
}

// deleteRole deletes a role by ID or name.
func (c *DeleteCmd) deleteRole(cl client.Client) error {
	var roleID uint

	if c.Name != nil {
		role, err := cl.Roles().GetByName(context.Background(), *c.Name)
		if err != nil {
			return fmt.Errorf("failed to get role by name: %w", err)
		}
		roleID = role.ID
	} else if c.ID != nil {
		parsedID, err := strconv.ParseUint(*c.ID, 10, 32)
		if err != nil {
			return fmt.Errorf("invalid ID format: %w", err)
		}
		roleID = uint(parsedID)
	}

	err := cl.Roles().Delete(context.Background(), roleID)
	if err != nil {
		return fmt.Errorf("failed to delete role: %w", err)
	}

	return nil
}
