package users

import (
	"context"
	"fmt"

	"github.com/input-output-hk/catalyst-forge/cli/cmd/cmds/api/auth/common"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/foundry/api/client"
)

type CreateCmd struct {
	Email  string `arg:"" help:"The email address of the user to create."`
	Status string `short:"s" help:"The status of the user (active, inactive)." default:"active"`
	JSON   bool   `short:"j" help:"Output as prettified JSON instead of table."`
}

func (c *CreateCmd) Run(ctx run.RunContext, cl client.Client) error {
	user, err := cl.CreateUser(context.Background(), &client.CreateUserRequest{
		Email:  c.Email,
		Status: c.Status,
	})
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	if c.JSON {
		return common.OutputUserJSON(user)
	}

	return common.OutputUserTable(user)
}
