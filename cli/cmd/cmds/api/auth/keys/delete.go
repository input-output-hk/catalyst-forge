package keys

import (
	"context"
	"fmt"
	"strconv"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/foundry/api/client"
)

type DeleteCmd struct {
	ID  *string `short:"i" help:"The numeric ID of the user key to delete (mutually exclusive with --kid)."`
	Kid *string `short:"k" help:"The key ID (KID) of the user key to delete (mutually exclusive with --id)."`
}

func (c *DeleteCmd) Run(ctx run.RunContext, cl client.Client) error {
	if c.ID == nil && c.Kid == nil {
		return fmt.Errorf("either --id or --kid must be specified")
	}

	if c.ID != nil && c.Kid != nil {
		return fmt.Errorf("only one of --id or --kid can be specified")
	}

	err := c.deleteUserKey(cl)
	if err != nil {
		return err
	}

	identifier := ""
	if c.ID != nil {
		identifier = *c.ID
	} else {
		identifier = *c.Kid
	}

	fmt.Printf("User key %s deleted successfully.\n", identifier)
	return nil
}

// deleteUserKey deletes a user key by ID or KID.
func (c *DeleteCmd) deleteUserKey(cl client.Client) error {
	var keyID uint

	if c.Kid != nil {
		userKey, err := cl.GetUserKeyByKid(context.Background(), *c.Kid)
		if err != nil {
			return fmt.Errorf("failed to get user key by KID: %w", err)
		}
		keyID = userKey.ID
	} else if c.ID != nil {
		parsedID, err := strconv.ParseUint(*c.ID, 10, 32)
		if err != nil {
			return fmt.Errorf("invalid ID format: %w", err)
		}
		keyID = uint(parsedID)
	}

	err := cl.DeleteUserKey(context.Background(), keyID)
	if err != nil {
		return fmt.Errorf("failed to delete user key: %w", err)
	}

	return nil
}
