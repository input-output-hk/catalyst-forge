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

type UpdateCmd struct {
	ID        *string `short:"i" help:"The numeric ID of the user key to update (mutually exclusive with --kid)."`
	Kid       *string `short:"k" help:"The key ID (KID) of the user key to update (mutually exclusive with --id)."`
	UserID    *string `short:"u" help:"The new user ID for the key."`
	NewKid    *string `short:"n" help:"The new KID for the key."`
	PubKeyB64 *string `short:"p" help:"The new base64-encoded public key."`
	Status    *string `short:"s" help:"The new status for the key (active, inactive)."`
	JSON      bool    `short:"j" help:"Output as prettified JSON instead of table."`
}

func (c *UpdateCmd) Run(ctx run.RunContext, cl client.Client) error {
	if c.ID == nil && c.Kid == nil {
		return fmt.Errorf("either --id or --kid must be specified")
	}

	if c.ID != nil && c.Kid != nil {
		return fmt.Errorf("only one of --id or --kid can be specified")
	}

	userKey, err := c.updateUserKey(cl)
	if err != nil {
		return err
	}

	if c.JSON {
		return common.OutputUserKeyJSON(userKey)
	}

	return common.OutputUserKeyTable(userKey)
}

// updateUserKey updates a user key by ID or KID.
func (c *UpdateCmd) updateUserKey(cl client.Client) (*users.UserKey, error) {
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

	req := &users.UpdateUserKeyRequest{}

	if c.UserID != nil {
		userID, err := strconv.ParseUint(*c.UserID, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid user ID format: %w", err)
		}
		userIDUint := uint(userID)
		req.UserID = &userIDUint
	}

	if c.NewKid != nil {
		req.Kid = c.NewKid
	}

	if c.PubKeyB64 != nil {
		req.PubKeyB64 = c.PubKeyB64
	}

	if c.Status != nil {
		req.Status = c.Status
	}

	userKey, err := cl.Keys().Update(context.Background(), keyID, req)
	if err != nil {
		return nil, fmt.Errorf("failed to update user key: %w", err)
	}

	return userKey, nil
}
