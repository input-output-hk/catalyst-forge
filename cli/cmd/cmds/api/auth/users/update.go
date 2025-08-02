package users

import (
	"context"
	"fmt"
	"strconv"

	"github.com/input-output-hk/catalyst-forge/cli/cmd/cmds/api/auth/common"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/foundry/api/client"
)

type UpdateCmd struct {
	ID     string `arg:"" help:"The ID of the user to update."`
	Email  string `short:"e" help:"The new email address for the user."`
	Status string `short:"s" help:"The new status for the user (active, inactive)."`
	JSON   bool   `short:"j" help:"Output as prettified JSON instead of table."`
}

func (c *UpdateCmd) Run(ctx run.RunContext, cl client.Client) error {
	// Convert string ID to uint
	id, err := strconv.ParseUint(c.ID, 10, 32)
	if err != nil {
		return fmt.Errorf("invalid ID format: %w", err)
	}

	req := &client.UpdateUserRequest{}

	if c.Email != "" {
		req.Email = c.Email
	}

	if c.Status != "" {
		req.Status = c.Status
	}

	user, err := cl.UpdateUser(context.Background(), uint(id), req)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	if c.JSON {
		return common.OutputUserJSON(user)
	}

	return common.OutputUserTable(user)
}
