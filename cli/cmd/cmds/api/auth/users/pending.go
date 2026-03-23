package users

import (
	"context"
	"fmt"

	"github.com/input-output-hk/catalyst-forge/cli/cmd/cmds/api/auth/common"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/client"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/client/users"
)

type PendingCmd struct {
	JSON bool `short:"j" help:"Output as prettified JSON instead of table."`
}

func (c *PendingCmd) Run(ctx run.RunContext, cl client.Client) error {
	users, err := c.getPendingUsers(cl)
	if err != nil {
		return err
	}

	if c.JSON {
		return common.OutputUsersJSON(users)
	}

	return common.OutputUsersTable(users)
}

// getPendingUsers retrieves all pending users.
func (c *PendingCmd) getPendingUsers(cl client.Client) ([]users.User, error) {
	users, err := cl.Users().GetPending(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get pending users: %w", err)
	}

	return users, nil
}
