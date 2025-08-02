package roles

import (
	"context"
	"fmt"
	"strconv"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/foundry/api/client"
)

type RemoveCmd struct {
	UserID    *string `short:"u" help:"The numeric ID of the user to remove (mutually exclusive with --user-email)."`
	UserEmail *string `short:"e" help:"The email of the user to remove (mutually exclusive with --user-id)."`
	RoleID    *string `short:"r" help:"The numeric ID of the role to remove from (mutually exclusive with --role-name)."`
	RoleName  *string `short:"n" help:"The name of the role to remove from (mutually exclusive with --role-id)."`
}

func (c *RemoveCmd) Run(ctx run.RunContext, cl client.Client) error {
	if c.UserID == nil && c.UserEmail == nil {
		return fmt.Errorf("either --user-id or --user-email must be specified")
	}

	if c.UserID != nil && c.UserEmail != nil {
		return fmt.Errorf("only one of --user-id or --user-email can be specified")
	}

	if c.RoleID == nil && c.RoleName == nil {
		return fmt.Errorf("either --role-id or --role-name must be specified")
	}

	if c.RoleID != nil && c.RoleName != nil {
		return fmt.Errorf("only one of --role-id or --role-name can be specified")
	}

	return c.removeUserFromRole(cl)
}

// removeUserFromRole removes a user from a role.
func (c *RemoveCmd) removeUserFromRole(cl client.Client) error {
	var userID uint

	if c.UserEmail != nil {
		user, err := cl.GetUserByEmail(context.Background(), *c.UserEmail)
		if err != nil {
			return fmt.Errorf("failed to find user with email %s: %w", *c.UserEmail, err)
		}
		userID = user.ID
	} else if c.UserID != nil {
		parsedID, err := strconv.ParseUint(*c.UserID, 10, 32)
		if err != nil {
			return fmt.Errorf("invalid user ID format: %w", err)
		}
		userID = uint(parsedID)
	}

	var roleID uint

	if c.RoleName != nil {
		role, err := cl.GetRoleByName(context.Background(), *c.RoleName)
		if err != nil {
			return fmt.Errorf("failed to find role with name %s: %w", *c.RoleName, err)
		}
		roleID = role.ID
	} else if c.RoleID != nil {
		parsedID, err := strconv.ParseUint(*c.RoleID, 10, 32)
		if err != nil {
			return fmt.Errorf("invalid role ID format: %w", err)
		}
		roleID = uint(parsedID)
	}

	if err := cl.RemoveUserFromRole(context.Background(), userID, roleID); err != nil {
		return fmt.Errorf("failed to remove user from role: %w", err)
	}

	fmt.Printf("Successfully removed user ID %d from role ID %d\n", userID, roleID)
	return nil
}
