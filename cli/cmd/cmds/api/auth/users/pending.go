package users

import (
	"context"
	"fmt"

	"github.com/input-output-hk/catalyst-forge/cli/cmd/cmds/api/auth/common"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/foundry/api/client"
)

type PendingCmd struct {
	Email *string `short:"e" help:"Optional email to get pending user keys for."`
	JSON  bool    `short:"j" help:"Output as prettified JSON instead of table."`
}

func (c *PendingCmd) Run(ctx run.RunContext, cl client.Client) error {
	// If email is provided, get pending user keys for that user
	if c.Email != nil {
		return c.getPendingKeysForUser(cl)
	}

	// Otherwise, get all pending users
	users, err := cl.GetPendingUsers(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get pending users: %w", err)
	}

	if c.JSON {
		return common.OutputUsersJSON(users)
	}

	return common.OutputUsersTable(users)
}

func (c *PendingCmd) getPendingKeysForUser(cl client.Client) error {
	// First, get the user by email
	user, err := cl.GetUserByEmail(context.Background(), *c.Email)
	if err != nil {
		return fmt.Errorf("failed to get user by email: %w", err)
	}

	// Then get all inactive keys for this user
	userKeys, err := cl.GetInactiveUserKeysByUserID(context.Background(), user.ID)
	if err != nil {
		return fmt.Errorf("failed to get inactive user keys: %w", err)
	}

	// Import the keys package to use its output functions
	if c.JSON {
		return common.OutputUserKeysJSON(userKeys)
	}

	return common.OutputUserKeysTable(userKeys)
}
