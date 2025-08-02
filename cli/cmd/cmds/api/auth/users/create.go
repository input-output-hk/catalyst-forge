package users

import (
	"context"
	"fmt"

	"github.com/input-output-hk/catalyst-forge/cli/cmd/cmds/api/auth/common"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/foundry/api/client"
)

type CreateCmd struct {
	Email  string `short:"e" help:"The email address of the user to create." required:"true"`
	Status string `short:"s" help:"The status of the user (active, inactive)." default:"active"`
	JSON   bool   `short:"j" help:"Output as prettified JSON instead of table."`
}

func (c *CreateCmd) Run(ctx run.RunContext, cl client.Client) error {
	user, err := c.createUser(cl)
	if err != nil {
		return err
	}

	if c.JSON {
		return common.OutputUserJSON(user)
	}

	return common.OutputUserTable(user)
}

// createUser creates a new user with the specified parameters.
func (c *CreateCmd) createUser(cl client.Client) (*client.User, error) {
	user, err := cl.CreateUser(context.Background(), &client.CreateUserRequest{
		Email:  c.Email,
		Status: c.Status,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}
