package keys

import (
	"context"
	"fmt"
	"strconv"

	"github.com/input-output-hk/catalyst-forge/cli/cmd/cmds/api/auth/common"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/foundry/api/client"
)

type CreateCmd struct {
	UserID    string `arg:"" help:"The ID of the user to create the key for."`
	Kid       string `arg:"" help:"The key ID (KID) for the user key."`
	PubKeyB64 string `arg:"" help:"The base64-encoded public key."`
	Status    string `short:"s" help:"The status of the user key (active, inactive)." default:"active"`
	JSON      bool   `short:"j" help:"Output as prettified JSON instead of table."`
}

func (c *CreateCmd) Run(ctx run.RunContext, cl client.Client) error {
	// Convert string UserID to uint
	userID, err := strconv.ParseUint(c.UserID, 10, 32)
	if err != nil {
		return fmt.Errorf("invalid user ID format: %w", err)
	}

	userKey, err := cl.CreateUserKey(context.Background(), &client.CreateUserKeyRequest{
		UserID:    uint(userID),
		Kid:       c.Kid,
		PubKeyB64: c.PubKeyB64,
		Status:    c.Status,
	})
	if err != nil {
		return fmt.Errorf("failed to create user key: %w", err)
	}

	if c.JSON {
		return common.OutputUserKeyJSON(userKey)
	}

	return common.OutputUserKeyTable(userKey)
}
