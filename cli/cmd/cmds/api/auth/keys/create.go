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

type CreateCmd struct {
	UserID    *string `short:"u" help:"The ID of the user to create the key for (mutually exclusive with --email)."`
	Email     *string `short:"e" help:"The email of the user to create the key for (mutually exclusive with --user-id)."`
	Kid       string  `short:"k" help:"The key ID (KID) for the user key." required:"true"`
	PubKeyB64 string  `short:"p" help:"The base64-encoded public key." required:"true"`
	Status    string  `short:"s" help:"The status of the user key (active, inactive)." default:"active"`
	JSON      bool    `short:"j" help:"Output as prettified JSON instead of table."`
}

func (c *CreateCmd) Run(ctx run.RunContext, cl client.Client) error {
	if c.UserID == nil && c.Email == nil {
		return fmt.Errorf("either --user-id or --email must be specified")
	}

	if c.UserID != nil && c.Email != nil {
		return fmt.Errorf("only one of --user-id or --email can be specified")
	}

	userKey, err := c.createUserKey(cl)
	if err != nil {
		return err
	}

	if c.JSON {
		return common.OutputUserKeyJSON(userKey)
	}

	return common.OutputUserKeyTable(userKey)
}

// createUserKey creates a user key for the given user.
func (c *CreateCmd) createUserKey(cl client.Client) (*users.UserKey, error) {
	var userID uint

	if c.Email != nil {
		user, err := cl.Users().GetByEmail(context.Background(), *c.Email)
		if err != nil {
			return nil, fmt.Errorf("failed to get user by email: %w", err)
		}
		userID = user.ID
	} else if c.UserID != nil {
		parsedUserID, err := strconv.ParseUint(*c.UserID, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid user ID format: %w", err)
		}
		userID = uint(parsedUserID)
	}

	userKey, err := cl.Keys().Create(context.Background(), &users.CreateUserKeyRequest{
		UserID:    userID,
		Kid:       c.Kid,
		PubKeyB64: c.PubKeyB64,
		Status:    c.Status,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create user key: %w", err)
	}

	return userKey, nil
}
