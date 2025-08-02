package roles

import (
	"context"
	"fmt"
	"strconv"

	"github.com/input-output-hk/catalyst-forge/cli/cmd/cmds/api/auth/common"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/foundry/api/client"
)

type UpdateCmd struct {
	ID          string   `arg:"" help:"The ID of the role to update."`
	Name        string   `short:"n" help:"The new name for the role."`
	Permissions []string `short:"p" help:"The new permissions for the role."`
	JSON        bool     `short:"j" help:"Output as prettified JSON instead of table."`
}

func (c *UpdateCmd) Run(ctx run.RunContext, cl client.Client) error {
	// Convert string ID to uint
	id, err := strconv.ParseUint(c.ID, 10, 32)
	if err != nil {
		return fmt.Errorf("invalid ID format: %w", err)
	}

	req := &client.UpdateRoleRequest{}

	if c.Name != "" {
		req.Name = c.Name
	}

	if len(c.Permissions) > 0 {
		req.Permissions = c.Permissions
	}

	role, err := cl.UpdateRole(context.Background(), uint(id), req)
	if err != nil {
		return fmt.Errorf("failed to update role: %w", err)
	}

	if c.JSON {
		return common.OutputRoleJSON(role)
	}

	return common.OutputRoleTable(role)
}
