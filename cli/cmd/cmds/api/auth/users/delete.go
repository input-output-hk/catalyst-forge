package users

import (
	"context"
	"fmt"
	"strconv"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/foundry/api/client/users"
)

type DeleteCmd struct {
	ID    *string `short:"i" help:"The numeric ID of the user to delete (mutually exclusive with --email)."`
	Email *string `short:"e" help:"The email of the user to delete (mutually exclusive with --id)."`
}

func (c *DeleteCmd) Run(ctx run.RunContext, cl interface{ Users() *users.UsersClient }) error {
	if c.ID == nil && c.Email == nil {
		return fmt.Errorf("either --id or --email must be specified")
	}

	if c.ID != nil && c.Email != nil {
		return fmt.Errorf("only one of --id or --email can be specified")
	}

	err := c.deleteUser(cl)
	if err != nil {
		return err
	}

	identifier := ""
	if c.ID != nil {
		identifier = *c.ID
	} else {
		identifier = *c.Email
	}

	fmt.Printf("User %s deleted successfully.\n", identifier)
	return nil
}

// deleteUser deletes a user by ID or email.
func (c *DeleteCmd) deleteUser(cl interface{ Users() *users.UsersClient }) error {
	var userID uint

	if c.Email != nil {
		user, err := cl.Users().GetByEmail(context.Background(), *c.Email)
		if err != nil {
			return fmt.Errorf("failed to get user by email: %w", err)
		}
		userID = user.ID
	} else if c.ID != nil {
		parsedID, err := strconv.ParseUint(*c.ID, 10, 32)
		if err != nil {
			return fmt.Errorf("invalid ID format: %w", err)
		}
		userID = uint(parsedID)
	}

	err := cl.Users().Delete(context.Background(), userID)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	return nil
}
