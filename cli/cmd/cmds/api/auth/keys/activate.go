package keys

import (
	"context"
	"fmt"
	"strconv"

	"github.com/input-output-hk/catalyst-forge/cli/cmd/cmds/api/auth/common"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/foundry/api/client"
)

type ActivateCmd struct {
	Kid    string  `arg:"" help:"The KID of the user key to activate."`
	UserID *string `short:"u" help:"The user ID that owns the key (mutually exclusive with --email)."`
	Email  *string `short:"e" help:"The email of the user that owns the key (mutually exclusive with --user-id)."`
	JSON   bool    `short:"j" help:"Output as prettified JSON instead of table."`
}

func (c *ActivateCmd) Run(ctx run.RunContext, cl client.Client) error {
	if c.UserID == nil && c.Email == nil {
		return fmt.Errorf("either --user-id or --email must be specified")
	}

	if c.UserID != nil && c.Email != nil {
		return fmt.Errorf("only one of --user-id or --email can be specified")
	}

	userKey, err := c.activateUserKey(cl)
	if err != nil {
		return err
	}

	if c.JSON {
		return common.OutputUserKeyJSON(userKey)
	}

	return common.OutputUserKeyTable(userKey)
}

// activateUserKey activates a user key by KID.
func (c *ActivateCmd) activateUserKey(cl client.Client) (*client.UserKey, error) {
	userKey, err := cl.GetUserKeyByKid(context.Background(), c.Kid)
	if err != nil {
		return nil, fmt.Errorf("failed to get user key by KID: %w", err)
	}

	if c.Email != nil {
		user, err := cl.GetUserByEmail(context.Background(), *c.Email)
		if err != nil {
			return nil, fmt.Errorf("failed to get user by email: %w", err)
		}

		if userKey.UserID != user.ID {
			return nil, fmt.Errorf("user key does not belong to the specified user")
		}
	} else if c.UserID != nil {
		userID, err := strconv.ParseUint(*c.UserID, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid user ID format: %w", err)
		}

		if userKey.UserID != uint(userID) {
			return nil, fmt.Errorf("user key does not belong to the specified user")
		}
	}

	req := &client.UpdateUserKeyRequest{
		UserID:    &userKey.UserID,
		Kid:       &userKey.Kid,
		PubKeyB64: &userKey.PubKeyB64,
		Status:    &[]string{"active"}[0],
	}

	updatedUserKey, err := cl.UpdateUserKey(context.Background(), userKey.ID, req)
	if err != nil {
		return nil, fmt.Errorf("failed to activate user key: %w", err)
	}

	return updatedUserKey, nil
}
