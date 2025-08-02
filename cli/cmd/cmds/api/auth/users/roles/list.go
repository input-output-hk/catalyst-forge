package roles

import (
	"context"
	"fmt"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/foundry/api/client"
)

type ListCmd struct {
	UserEmail string `arg:"" optional:"" help:"The email of the user to list roles for."`
	RoleName  string `arg:"" optional:"" help:"The name of the role to list users for."`
	JSON      bool   `short:"j" help:"Output as prettified JSON instead of table."`
}

func (c *ListCmd) Run(ctx run.RunContext, cl client.Client) error {
	if c.UserEmail != "" && c.RoleName != "" {
		return fmt.Errorf("cannot specify both user email and role name")
	}

	if c.UserEmail == "" && c.RoleName == "" {
		return fmt.Errorf("must specify either user email or role name")
	}

	if c.UserEmail != "" {
		return c.listUserRoles(cl)
	}

	return c.listRoleUsers(cl)
}

func (c *ListCmd) listUserRoles(cl client.Client) error {
	// Look up user by email
	user, err := cl.GetUserByEmail(context.Background(), c.UserEmail)
	if err != nil {
		return fmt.Errorf("failed to find user with email %s: %w", c.UserEmail, err)
	}

	userRoles, err := cl.GetUserRoles(context.Background(), user.ID)
	if err != nil {
		return fmt.Errorf("failed to get user roles: %w", err)
	}

	if c.JSON {
		// TODO: Implement JSON output
		fmt.Printf("User roles for user %s: %+v\n", c.UserEmail, userRoles)
		return nil
	}

	fmt.Printf("Roles for user %s:\n", c.UserEmail)
	for _, userRole := range userRoles {
		fmt.Printf("  - Role ID: %d, Created: %s\n", userRole.RoleID, userRole.CreatedAt.Format("2006-01-02 15:04:05"))
	}

	return nil
}

func (c *ListCmd) listRoleUsers(cl client.Client) error {
	// Look up role by name
	role, err := cl.GetRoleByName(context.Background(), c.RoleName)
	if err != nil {
		return fmt.Errorf("failed to find role with name %s: %w", c.RoleName, err)
	}

	userRoles, err := cl.GetRoleUsers(context.Background(), role.ID)
	if err != nil {
		return fmt.Errorf("failed to get role users: %w", err)
	}

	if c.JSON {
		// TODO: Implement JSON output
		fmt.Printf("Role users for role %s: %+v\n", c.RoleName, userRoles)
		return nil
	}

	fmt.Printf("Users for role %s:\n", c.RoleName)
	for _, userRole := range userRoles {
		fmt.Printf("  - User ID: %d, Created: %s\n", userRole.UserID, userRole.CreatedAt.Format("2006-01-02 15:04:05"))
	}

	return nil
}
