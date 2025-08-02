package roles

import (
	"context"
	"fmt"
	"strconv"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/foundry/api/client"
)

type DeleteCmd struct {
	ID string `arg:"" help:"The ID of the role to delete."`
}

func (c *DeleteCmd) Run(ctx run.RunContext, cl client.Client) error {
	// Convert string ID to uint
	id, err := strconv.ParseUint(c.ID, 10, 32)
	if err != nil {
		return fmt.Errorf("invalid ID format: %w", err)
	}

	err = cl.DeleteRole(context.Background(), uint(id))
	if err != nil {
		return fmt.Errorf("failed to delete role: %w", err)
	}

	fmt.Printf("Role %s deleted successfully.\n", c.ID)
	return nil
}
