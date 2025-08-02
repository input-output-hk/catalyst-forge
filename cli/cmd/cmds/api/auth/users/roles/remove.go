package roles

import (
	"context"
	"fmt"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/foundry/api/client"
)

type RemoveCmd struct {
	UserEmail string `arg:"" help:"The email of the user to remove from the role."`
	RoleName  string `arg:"" help:"The name of the role to remove the user from."`
}

func (c *RemoveCmd) Run(ctx run.RunContext, cl client.Client) error {
	// Look up user by email
	user, err := cl.GetUserByEmail(context.Background(), c.UserEmail)
	if err != nil {
		return fmt.Errorf("failed to find user with email %s: %w", c.UserEmail, err)
	}

	// Look up role by name
	role, err := cl.GetRoleByName(context.Background(), c.RoleName)
	if err != nil {
		return fmt.Errorf("failed to find role with name %s: %w", c.RoleName, err)
	}

	if err := cl.RemoveUserFromRole(context.Background(), user.ID, role.ID); err != nil {
		return fmt.Errorf("failed to remove user from role: %w", err)
	}

	fmt.Printf("Successfully removed user %s from role %s\n", c.UserEmail, c.RoleName)
	return nil
}
