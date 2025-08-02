package users

import (
	"context"
	"fmt"
	"strconv"

	"github.com/input-output-hk/catalyst-forge/cli/cmd/cmds/api/auth/common"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/foundry/api/client/users"
)

type ActivateCmd struct {
	ID    *string `short:"i" help:"The ID of the user to activate (mutually exclusive with --email)."`
	Email *string `short:"e" help:"The email of the user to activate (mutually exclusive with --id)."`
	JSON  bool    `short:"j" help:"Output as prettified JSON instead of table."`
}

func (c *ActivateCmd) Run(ctx run.RunContext, cl interface{ Users() *users.UsersClient }) error {
	if c.ID == nil && c.Email == nil {
		return fmt.Errorf("either --id or --email must be specified")
	}

	if c.ID != nil && c.Email != nil {
		return fmt.Errorf("only one of --id or --email can be specified")
	}

	user, err := c.activateUser(cl)
	if err != nil {
		return err
	}

	if c.JSON {
		return common.OutputUserJSON(user)
	}

	return common.OutputUserTable(user)
}

// activateUser activates a user by ID or email.
func (c *ActivateCmd) activateUser(cl interface{ Users() *users.UsersClient }) (*users.User, error) {
	if c.ID != nil {
		id, err := strconv.ParseUint(*c.ID, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid ID format: %w", err)
		}
		user, err := cl.Users().Activate(context.Background(), uint(id))
		if err != nil {
			return nil, fmt.Errorf("failed to activate user by ID: %w", err)
		}
		return user, nil
	}

	user, err := cl.Users().GetByEmail(context.Background(), *c.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	activatedUser, err := cl.Users().Activate(context.Background(), user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to activate user: %w", err)
	}
	return activatedUser, nil
}
