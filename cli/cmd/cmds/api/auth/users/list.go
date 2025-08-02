package users

import (
	"context"
	"fmt"

	"github.com/input-output-hk/catalyst-forge/cli/cmd/cmds/api/auth/common"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/foundry/api/client"
)

type ListCmd struct {
	JSON bool `short:"j" help:"Output as prettified JSON instead of table."`
}

func (c *ListCmd) Run(ctx run.RunContext, cl client.Client) error {
	users, err := cl.ListUsers(context.Background())
	if err != nil {
		return fmt.Errorf("failed to list users: %w", err)
	}

	if c.JSON {
		return common.OutputUsersJSON(users)
	}

	return common.OutputUsersTable(users)
}
