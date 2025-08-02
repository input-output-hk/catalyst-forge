package roles

import (
	"context"
	"fmt"
	"strconv"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/foundry/api/client/users"
)

type ListCmd struct {
	UserID    *string `short:"u" help:"The numeric ID of the user to list roles for (mutually exclusive with --user-email, --role-id, --role-name)."`
	UserEmail *string `short:"e" help:"The email of the user to list roles for (mutually exclusive with --user-id, --role-id, --role-name)."`
	RoleID    *string `short:"r" help:"The numeric ID of the role to list users for (mutually exclusive with --user-id, --user-email, --role-name)."`
	RoleName  *string `short:"n" help:"The name of the role to list users for (mutually exclusive with --user-id, --user-email, --role-id)."`
	JSON      bool    `short:"j" help:"Output as prettified JSON instead of table."`
}

func (c *ListCmd) Run(ctx run.RunContext, cl interface {
	Users() *users.UsersClient
	Roles() *users.RolesClient
}) error {
	userSpecified := c.UserID != nil || c.UserEmail != nil
	roleSpecified := c.RoleID != nil || c.RoleName != nil

	if !userSpecified && !roleSpecified {
		return fmt.Errorf("must specify either user (--user-id or --user-email) or role (--role-id or --role-name)")
	}

	if userSpecified && roleSpecified {
		return fmt.Errorf("cannot specify both user and role")
	}

	if userSpecified {
		return c.listUserRoles(cl)
	}

	return c.listRoleUsers(cl)
}

// listUserRoles lists roles for a specific user.
func (c *ListCmd) listUserRoles(cl interface {
	Users() *users.UsersClient
	Roles() *users.RolesClient
}) error {
	var userID uint

	if c.UserEmail != nil {
		user, err := cl.Users().GetByEmail(context.Background(), *c.UserEmail)
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

	userRoles, err := cl.Roles().GetUserRoles(context.Background(), userID)
	if err != nil {
		return fmt.Errorf("failed to get user roles: %w", err)
	}

	if c.JSON {
		fmt.Printf("User roles for user ID %d: %+v\n", userID, userRoles)
		return nil
	}

	fmt.Printf("Roles for user ID %d:\n", userID)
	for _, userRole := range userRoles {
		fmt.Printf("  - Role ID: %d, Created: %s\n", userRole.RoleID, userRole.CreatedAt.Format("2006-01-02 15:04:05"))
	}

	return nil
}

// listRoleUsers lists users for a specific role.
func (c *ListCmd) listRoleUsers(cl interface {
	Users() *users.UsersClient
	Roles() *users.RolesClient
}) error {
	var roleID uint

	if c.RoleName != nil {
		role, err := cl.Roles().GetByName(context.Background(), *c.RoleName)
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

	userRoles, err := cl.Roles().GetRoleUsers(context.Background(), roleID)
	if err != nil {
		return fmt.Errorf("failed to get role users: %w", err)
	}

	if c.JSON {
		fmt.Printf("Role users for role ID %d: %+v\n", roleID, userRoles)
		return nil
	}

	fmt.Printf("Users for role ID %d:\n", roleID)
	for _, userRole := range userRoles {
		fmt.Printf("  - User ID: %d, Created: %s\n", userRole.UserID, userRole.CreatedAt.Format("2006-01-02 15:04:05"))
	}

	return nil
}
