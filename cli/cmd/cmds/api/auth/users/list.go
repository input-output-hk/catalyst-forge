package users

import (
	"context"
	"fmt"

	"github.com/input-output-hk/catalyst-forge/cli/cmd/cmds/api/auth/common"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/foundry/api/client/users"
)

type ListCmd struct {
	JSON bool `short:"j" help:"Output as prettified JSON instead of table."`
}

func (c *ListCmd) Run(ctx run.RunContext, cl interface{ Users() *users.UsersClient }) error {
	users, err := c.listUsers(cl)
	if err != nil {
		return err
	}

	if c.JSON {
		return common.OutputUsersJSON(users)
	}

	return common.OutputUsersTable(users)
}

// listUsers retrieves all users.
func (c *ListCmd) listUsers(cl interface{ Users() *users.UsersClient }) ([]users.User, error) {
	users, err := cl.Users().List(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	return users, nil
}
