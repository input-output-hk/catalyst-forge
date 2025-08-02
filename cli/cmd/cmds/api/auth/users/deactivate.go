package users

import (
	"context"
	"fmt"
	"strconv"

	"github.com/input-output-hk/catalyst-forge/cli/cmd/cmds/api/auth/common"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/foundry/api/client/users"
)

type DeactivateCmd struct {
	ID    *string `short:"i" help:"The numeric ID of the user to deactivate (mutually exclusive with --email)."`
	Email *string `short:"e" help:"The email of the user to deactivate (mutually exclusive with --id)."`
	JSON  bool    `short:"j" help:"Output as prettified JSON instead of table."`
}

func (c *DeactivateCmd) Run(ctx run.RunContext, cl interface{ Users() *users.UsersClient }) error {
	if c.ID == nil && c.Email == nil {
		return fmt.Errorf("either --id or --email must be specified")
	}

	if c.ID != nil && c.Email != nil {
		return fmt.Errorf("only one of --id or --email can be specified")
	}

	user, err := c.deactivateUser(cl)
	if err != nil {
		return err
	}

	if c.JSON {
		return common.OutputUserJSON(user)
	}

	return common.OutputUserTable(user)
}

// deactivateUser deactivates a user by ID or email.
func (c *DeactivateCmd) deactivateUser(cl interface{ Users() *users.UsersClient }) (*users.User, error) {
	var userID uint

	if c.Email != nil {
		user, err := cl.Users().GetByEmail(context.Background(), *c.Email)
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

	user, err := cl.Users().Deactivate(context.Background(), userID)
	if err != nil {
		return nil, fmt.Errorf("failed to deactivate user: %w", err)
	}

	return user, nil
}
