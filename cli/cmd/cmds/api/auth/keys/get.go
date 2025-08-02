package keys

import (
	"context"
	"fmt"
	"strconv"

	"github.com/input-output-hk/catalyst-forge/cli/cmd/cmds/api/auth/common"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/foundry/api/client"
)

type GetCmd struct {
	ID     *string `short:"i" help:"The ID of the user key to retrieve (mutually exclusive with --kid and --user-id)."`
	Kid    *string `short:"k" help:"The KID of the user key to retrieve (mutually exclusive with --id and --user-id)."`
	UserID *string `short:"u" help:"The user ID to get keys for (mutually exclusive with --id and --kid)."`
	Active bool    `short:"a" help:"Only show active keys when getting by user ID."`
	JSON   bool    `short:"j" help:"Output as prettified JSON instead of table."`
}

func (c *GetCmd) Run(ctx run.RunContext, cl client.Client) error {
	if c.UserID != nil {
		return c.getByUser(cl)
	}

	if c.ID == nil && c.Kid == nil {
		return fmt.Errorf("either --id, --kid, or --user-id must be specified")
	}

	if c.ID != nil && c.Kid != nil {
		return fmt.Errorf("only one of --id or --kid can be specified")
	}

	userKey, err := c.retrieveUserKey(cl)
	if err != nil {
		return err
	}

	if c.JSON {
		return common.OutputUserKeyJSON(userKey)
	}

	return common.OutputUserKeyTable(userKey)
}

// retrieveUserKey retrieves a user key by ID or KID.
func (c *GetCmd) retrieveUserKey(cl client.Client) (*client.UserKey, error) {
	if c.ID != nil {
		id, err := strconv.ParseUint(*c.ID, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid ID format: %w", err)
		}
		userKey, err := cl.GetUserKey(context.Background(), uint(id))
		if err != nil {
			return nil, fmt.Errorf("failed to get user key by ID: %w", err)
		}
		return userKey, nil
	}

	userKey, err := cl.GetUserKeyByKid(context.Background(), *c.Kid)
	if err != nil {
		return nil, fmt.Errorf("failed to get user key by KID: %w", err)
	}
	return userKey, nil
}

// getByUser retrieves all user keys for a given user ID.
func (c *GetCmd) getByUser(cl client.Client) error {
	userID, err := strconv.ParseUint(*c.UserID, 10, 32)
	if err != nil {
		return fmt.Errorf("invalid user ID format: %w", err)
	}

	var userKeys []client.UserKey

	if c.Active {
		userKeys, err = cl.GetActiveUserKeysByUserID(context.Background(), uint(userID))
		if err != nil {
			return fmt.Errorf("failed to get active user keys: %w", err)
		}
	} else {
		userKeys, err = cl.GetUserKeysByUserID(context.Background(), uint(userID))
		if err != nil {
			return fmt.Errorf("failed to get user keys: %w", err)
		}
	}

	if c.JSON {
		return common.OutputUserKeysJSON(userKeys)
	}

	return common.OutputUserKeysTable(userKeys)
}
