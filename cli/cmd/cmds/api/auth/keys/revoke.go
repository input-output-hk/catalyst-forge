package keys

import (
	"context"
	"fmt"
	"strconv"

	"github.com/input-output-hk/catalyst-forge/cli/cmd/cmds/api/auth/common"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/foundry/api/client"
	"github.com/input-output-hk/catalyst-forge/foundry/api/client/users"
)

type RevokeCmd struct {
	ID   *string `short:"i" help:"The numeric ID of the user key to revoke (mutually exclusive with --kid)."`
	Kid  *string `short:"k" help:"The key ID (KID) of the user key to revoke (mutually exclusive with --id)."`
	JSON bool    `short:"j" help:"Output as prettified JSON instead of table."`
}

func (c *RevokeCmd) Run(ctx run.RunContext, cl client.Client) error {
	if c.ID == nil && c.Kid == nil {
		return fmt.Errorf("either --id or --kid must be specified")
	}

	if c.ID != nil && c.Kid != nil {
		return fmt.Errorf("only one of --id or --kid can be specified")
	}

	userKey, err := c.revokeUserKey(cl)
	if err != nil {
		return err
	}

	if c.JSON {
		return common.OutputUserKeyJSON(userKey)
	}

	return common.OutputUserKeyTable(userKey)
}

// revokeUserKey revokes a user key by ID or KID.
func (c *RevokeCmd) revokeUserKey(cl client.Client) (*users.UserKey, error) {
	var keyID uint

	if c.Kid != nil {
		userKey, err := cl.Keys().GetByKid(context.Background(), *c.Kid)
		if err != nil {
			return nil, fmt.Errorf("failed to get user key by KID: %w", err)
		}
		keyID = userKey.ID
	} else if c.ID != nil {
		parsedID, err := strconv.ParseUint(*c.ID, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid ID format: %w", err)
		}
		keyID = uint(parsedID)
	}

	userKey, err := cl.Keys().Revoke(context.Background(), keyID)
	if err != nil {
		return nil, fmt.Errorf("failed to revoke user key: %w", err)
	}

	return userKey, nil
}
