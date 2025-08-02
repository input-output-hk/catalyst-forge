package keys

import (
	"context"
	"fmt"

	"github.com/input-output-hk/catalyst-forge/cli/cmd/cmds/api/auth/common"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/foundry/api/client"
)

type PendingCmd struct {
	JSON bool `short:"j" help:"Output as prettified JSON instead of table."`
}

func (c *PendingCmd) Run(ctx run.RunContext, cl client.Client) error {
	userKeys, err := cl.GetInactiveUserKeys(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get inactive user keys: %w", err)
	}

	if c.JSON {
		return common.OutputUserKeysJSON(userKeys)
	}

	return common.OutputUserKeysTable(userKeys)
}
