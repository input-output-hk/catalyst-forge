package api

import (
	"context"
	"fmt"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/config"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/foundry/api/client"
)

type LoginCmd struct {
	Token string `arg:"" help:"The token to use for authentication." env:"FOUNDRY_TOKEN"`
	Type  string `short:"t" help:"The type of login to perform." enum:"gha,foundry" default:"foundry"`
}

func (c *LoginCmd) Run(ctx run.RunContext, cl client.Client) error {
	var jwt string
	switch c.Type {
	case "gha":
		resp, err := cl.ValidateToken(context.Background(), &client.ValidateTokenRequest{
			Token: c.Token,
		})
		if err != nil {
			return fmt.Errorf("failed to login: %w", err)
		}

		jwt = resp.Token
	case "foundry":
		jwt = c.Token
	default:
		return fmt.Errorf("invalid login type: %s", c.Type)
	}

	if ctx.Config == nil {
		ctx.Config = config.NewCustomConfig(ctx.FS)
	}

	configPath, err := ctx.Config.ConfigPath()
	if err != nil {
		return fmt.Errorf("failed to get config path: %w", err)
	}

	ctx.Logger.Info("Login successful, saving token", "path", configPath)
	ctx.Config.Token = jwt
	if err := ctx.Config.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	ctx.Logger.Info("Token saved", "path", configPath)
	return nil
}
