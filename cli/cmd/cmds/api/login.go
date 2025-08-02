package api

import (
	"context"
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/foundry/api/client/auth"
	"github.com/input-output-hk/catalyst-forge/foundry/api/client/gha"
	authpkg "github.com/input-output-hk/catalyst-forge/foundry/api/pkg/auth"
)

type LoginCmd struct {
	Email string `short:"e" help:"The email of the user to login as."`
	Token string `help:"An existing JWT token to use for authentication." env:"FOUNDRY_TOKEN"`
	Type  string `short:"t" help:"The type of login to perform." enum:"gha,foundry" default:"foundry"`
}

type EmailForm struct {
	Email string `form:"email"`
}

func (c *LoginCmd) Run(ctx run.RunContext, cl interface {
	GHA() *gha.GHAClient
	Auth() *auth.AuthClient
}) error {
	var jwt string
	var form EmailForm

	manager := authpkg.NewAuthManager(authpkg.WithFilesystem(ctx.FS))

	emailFlow := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Work email").
				Description("Enter your work email address").
				Placeholder("user@company.com").
				Value(&form.Email).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("email is required")
					}
					// Basic email validation
					if len(s) < 5 || !contains(s, "@") {
						return fmt.Errorf("please enter a valid email address")
					}
					return nil
				}),
		),
	)

	switch c.Type {
	case "gha":
		resp, err := cl.GHA().ValidateToken(context.Background(), &gha.ValidateTokenRequest{
			Token: c.Token,
		})
		if err != nil {
			return fmt.Errorf("failed to login: %w", err)
		}

		jwt = resp.Token
	case "foundry":
		if c.Token != "" {
			jwt = c.Token
		} else {
			var email string
			if ctx.Config.Email == "" {
				if err := emailFlow.Run(); err != nil {
					return fmt.Errorf("failed to run email form: %w", err)
				}

				ctx.Config.Email = form.Email
				if err := ctx.Config.Save(); err != nil {
					return fmt.Errorf("failed to save config: %w", err)
				}
				email = form.Email
			} else {
				email = ctx.Config.Email
			}

			stateDir, err := getStateDir(ctx)
			if err != nil {
				return fmt.Errorf("failed to get state directory: %w", err)
			}

			ctx.Logger.Debug("Loading key pair", "stateDir", stateDir)
			kp, err := manager.LoadKeyPair(stateDir)
			if err != nil {
				return fmt.Errorf("failed to load key pair: %w", err)
			}

			ctx.Logger.Debug("Creating challenge", "email", email, "kid", kp.Kid())
			challenge, err := cl.Auth().CreateChallenge(context.Background(), &auth.ChallengeRequest{
				Email: email,
				Kid:   kp.Kid(),
			})
			if err != nil {
				return fmt.Errorf("failed to create challenge: %w", err)
			}

			ctx.Logger.Debug("Signing challenge", "challenge", challenge)
			challengeResponse, err := kp.SignChallenge(challenge)
			if err != nil {
				return fmt.Errorf("failed to sign challenge: %w", err)
			}

			ctx.Logger.Debug("Logging in", "challengeResponse", challengeResponse)
			resp, err := cl.Auth().Login(context.Background(), challengeResponse)
			if err != nil {
				return fmt.Errorf("failed to login: %w", err)
			}

			jwt = resp.Token
		}
	default:
		return fmt.Errorf("invalid login type: %s", c.Type)
	}

	ctx.Logger.Info("Login successful, saving token")
	ctx.Config.Token = jwt
	if err := ctx.Config.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	ctx.Logger.Info("Token saved")
	return nil
}
