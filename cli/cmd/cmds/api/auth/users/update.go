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
	ID       *string `short:"i" help:"The numeric ID of the user to update (mutually exclusive with --email)."`
	Email    *string `short:"e" help:"The email of the user to update (mutually exclusive with --id)."`
	NewEmail *string `short:"n" help:"The new email address for the user."`
	Status   *string `short:"s" help:"The new status for the user (active, inactive)."`
	JSON     bool    `short:"j" help:"Output as prettified JSON instead of table."`
}

func (c *UpdateCmd) Run(ctx run.RunContext, cl client.Client) error {
	if c.ID == nil && c.Email == nil {
		return fmt.Errorf("either --id or --email must be specified")
	}

	if c.ID != nil && c.Email != nil {
		return fmt.Errorf("only one of --id or --email can be specified")
	}

	user, err := c.updateUser(cl)
	if err != nil {
		return err
	}

	if c.JSON {
		return common.OutputUserJSON(user)
	}

	return common.OutputUserTable(user)
}

// updateUser updates a user by ID or email.
func (c *UpdateCmd) updateUser(cl client.Client) (*client.User, error) {
	var userID uint

	if c.Email != nil {
		user, err := cl.GetUserByEmail(context.Background(), *c.Email)
		if err != nil {
			return nil, fmt.Errorf("failed to get user by email: %w", err)
		}
		userID = user.ID
	} else if c.ID != nil {
		parsedID, err := strconv.ParseUint(*c.ID, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid ID format: %w", err)
		}
		userID = uint(parsedID)
	}

	req := &client.UpdateUserRequest{}

	if c.NewEmail != nil {
		req.Email = *c.NewEmail
	}

	if c.Status != nil {
		req.Status = *c.Status
	}

	user, err := cl.UpdateUser(context.Background(), userID, req)
	if err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return user, nil
}
