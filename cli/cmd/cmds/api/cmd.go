package api

import (
	"fmt"

	"github.com/alecthomas/kong"
	"github.com/input-output-hk/catalyst-forge/cli/cmd/cmds/api/auth"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/utils"
	"github.com/input-output-hk/catalyst-forge/foundry/api/client"
)

type ApiCmd struct {
	Auth     auth.AuthCmd `cmd:"" help:"Manage API authentication."`
	Login    LoginCmd     `cmd:"" help:"Login to the Foundry API."`
	Register RegisterCmd  `cmd:"" help:"Register a new user with the Foundry API."`
}

func (c *ApiCmd) AfterApply(kctx *kong.Context, ctx run.RunContext) error {
	cl, err := utils.NewAPIClient(ctx.RootProject, ctx)
	if err != nil {
		return fmt.Errorf("cannot create API client: %w", err)
	}

	kctx.BindTo(cl, (*client.Client)(nil))
	return nil
}
