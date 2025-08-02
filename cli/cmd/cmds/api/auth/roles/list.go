package roles

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
	roles, err := cl.ListRoles(context.Background())
	if err != nil {
		return fmt.Errorf("failed to list roles: %w", err)
	}

	if c.JSON {
		return common.OutputRolesJSON(roles)
	}

	return common.OutputRolesTable(roles)
}
