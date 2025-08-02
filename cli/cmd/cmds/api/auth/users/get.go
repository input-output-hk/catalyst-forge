package users

import (
	"context"
	"fmt"
	"strconv"

	"github.com/input-output-hk/catalyst-forge/cli/cmd/cmds/api/auth/common"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/foundry/api/client"
)

type GetCmd struct {
	ID    *string `short:"i" help:"The ID of the user to retrieve."`
	Email *string `short:"e" help:"The email of the user to retrieve."`
	JSON  bool    `short:"j" help:"Output as prettified JSON instead of table."`
}

func (c *GetCmd) Run(ctx run.RunContext, cl client.Client) error {
	if c.ID == nil && c.Email == nil {
		return fmt.Errorf("either --id or --email must be specified")
	}

	if c.ID != nil && c.Email != nil {
		return fmt.Errorf("only one of --id or --email can be specified")
	}

	user, err := c.retrieveUser(cl)
	if err != nil {
		return err
	}

	if c.JSON {
		return common.OutputUserJSON(user)
	}

	return common.OutputUserTable(user)
}

func (c *GetCmd) retrieveUser(cl client.Client) (*client.User, error) {
	if c.ID != nil {
		// Convert string ID to uint
		id, err := strconv.ParseUint(*c.ID, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid ID format: %w", err)
		}
		user, err := cl.GetUser(context.Background(), uint(id))
		if err != nil {
			return nil, fmt.Errorf("failed to get user by ID: %w", err)
		}
		return user, nil
	}

	user, err := cl.GetUserByEmail(context.Background(), *c.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}
	return user, nil
}
