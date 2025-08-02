package keys

import (
	"context"
	"fmt"
	"strconv"

	"github.com/input-output-hk/catalyst-forge/cli/cmd/cmds/api/auth/common"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/foundry/api/client"
)

type RevokeCmd struct {
	ID   string `arg:"" help:"The ID of the user key to revoke."`
	JSON bool   `short:"j" help:"Output as prettified JSON instead of table."`
}

func (c *RevokeCmd) Run(ctx run.RunContext, cl client.Client) error {
	// Convert string ID to uint
	id, err := strconv.ParseUint(c.ID, 10, 32)
	if err != nil {
		return fmt.Errorf("invalid ID format: %w", err)
	}

	userKey, err := cl.RevokeUserKey(context.Background(), uint(id))
	if err != nil {
		return fmt.Errorf("failed to revoke user key: %w", err)
	}

	if c.JSON {
		return common.OutputUserKeyJSON(userKey)
	}

	return common.OutputUserKeyTable(userKey)
}
