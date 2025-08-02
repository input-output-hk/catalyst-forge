package keys

import (
	"context"
	"fmt"
	"strconv"

	"github.com/input-output-hk/catalyst-forge/cli/cmd/cmds/api/auth/common"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/foundry/api/client"
)

type UpdateCmd struct {
	ID        string `arg:"" help:"The ID of the user key to update."`
	UserID    string `short:"u" help:"The new user ID for the key."`
	Kid       string `short:"k" help:"The new KID for the key."`
	PubKeyB64 string `short:"p" help:"The new base64-encoded public key."`
	Status    string `short:"s" help:"The new status for the key (active, inactive)."`
	JSON      bool   `short:"j" help:"Output as prettified JSON instead of table."`
}

func (c *UpdateCmd) Run(ctx run.RunContext, cl client.Client) error {
	// Convert string ID to uint
	id, err := strconv.ParseUint(c.ID, 10, 32)
	if err != nil {
		return fmt.Errorf("invalid ID format: %w", err)
	}

	req := &client.UpdateUserKeyRequest{}

	if c.UserID != "" {
		// Convert string UserID to uint
		userID, err := strconv.ParseUint(c.UserID, 10, 32)
		if err != nil {
			return fmt.Errorf("invalid user ID format: %w", err)
		}
		req.UserID = uint(userID)
	}

	if c.Kid != "" {
		req.Kid = c.Kid
	}

	if c.PubKeyB64 != "" {
		req.PubKeyB64 = c.PubKeyB64
	}

	if c.Status != "" {
		req.Status = c.Status
	}

	userKey, err := cl.UpdateUserKey(context.Background(), uint(id), req)
	if err != nil {
		return fmt.Errorf("failed to update user key: %w", err)
	}

	if c.JSON {
		return common.OutputUserKeyJSON(userKey)
	}

	return common.OutputUserKeyTable(userKey)
}
