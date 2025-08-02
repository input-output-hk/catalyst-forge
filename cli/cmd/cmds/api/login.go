package api

import (
	"context"
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/input-output-hk/catalyst-forge/cli/internal/state"
	"github.com/input-output-hk/catalyst-forge/cli/internal/ux"
	"github.com/input-output-hk/catalyst-forge/cli/internal/validator"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/foundry/api/client"
	"github.com/input-output-hk/catalyst-forge/foundry/api/client/auth"
	"github.com/input-output-hk/catalyst-forge/foundry/api/client/github"
	authpkg "github.com/input-output-hk/catalyst-forge/foundry/api/pkg/auth"
)

type LoginCmd struct {
	Email string `short:"e" help:"The email of the user to login as."`
	Token string `help:"An existing JWT token to use for authentication." env:"FOUNDRY_TOKEN"`
	Type  string `short:"t" help:"The type of login to perform." enum:"github,foundry" default:"foundry"`
}

type EmailForm struct {
	Email string `form:"email"`
}

func (c *LoginCmd) Run(ctx run.RunContext, cl client.Client) error {
	var jwt string
	var err error

	switch c.Type {
	case "github":
		var resp *github.ValidateTokenResponse
		err = ux.NewSpinner().
			Title("Validating GitHub token...").
			Action(func() {
				resp, err = cl.Github().ValidateToken(context.Background(), &github.ValidateTokenRequest{
					Token: c.Token,
				})
			}).Run()
		if err != nil {
			return fmt.Errorf("failed to login with github: %w", err)
		}
		jwt = resp.Token

	case "foundry":
		if c.Token != "" {
			jwt = c.Token
		} else {
			jwt, err = c.interactiveFoundryLogin(ctx, cl)
			if err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("invalid login type: %s", c.Type)
	}

	err = ux.NewSpinner().
		Title("Saving token...").
		Action(func() {
			ctx.Config.Token = jwt
			err = ctx.Config.Save()
		}).Run()
	if err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	ux.Success("Login successful!")
	return nil
}

func (c *LoginCmd) interactiveFoundryLogin(ctx run.RunContext, cl client.Client) (string, error) {
	var form EmailForm
	var email string
	var err error

	if ctx.Config.Email == "" {
		emailForm := ux.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Work email").
					Description("Enter your work email address").
					Placeholder("user@company.com").
					Value(&form.Email).
					Validate(validator.Email),
			),
		)
		if err = emailForm.Run(); err != nil {
			return "", fmt.Errorf("failed to run email form: %w", err)
		}
		email = form.Email
		// Save email to config for future use
		err = ux.NewSpinner().
			Title("Saving email to config...").
			Action(func() {
				ctx.Config.Email = email
				err = ctx.Config.Save()
			}).Run()
		if err != nil {
			return "", fmt.Errorf("failed to save config: %w", err)
		}
	} else {
		email = ctx.Config.Email
	}

	manager := authpkg.NewAuthManager(authpkg.WithFilesystem(ctx.FS))
	stateDir, err := state.GetDir(ctx)
	if err != nil {
		return "", err
	}

	var kp *authpkg.KeyPair
	err = ux.NewSpinner().
		Title("Loading key pair...").
		Action(func() {
			kp, err = manager.LoadKeyPair(stateDir)
		}).Run()
	if err != nil {
		return "", fmt.Errorf("failed to load key pair: %w", err)
	}

	var challenge *authpkg.KeyPairChallenge
	err = ux.NewSpinner().
		Title("Requesting login challenge...").
		Action(func() {
			challenge, err = cl.Auth().CreateChallenge(context.Background(), &auth.ChallengeRequest{
				Email: email,
				Kid:   kp.Kid(),
			})
		}).Run()
	if err != nil {
		return "", fmt.Errorf("failed to create challenge: %w", err)
	}

	var challengeResponse *authpkg.KeyPairChallengeResponse
	err = ux.NewSpinner().
		Title("Signing login challenge...").
		Action(func() {
			challengeResponse, err = kp.SignChallenge(challenge)
		}).Run()
	if err != nil {
		return "", fmt.Errorf("failed to sign challenge: %w", err)
	}

	var resp *auth.LoginResponse
	err = ux.NewSpinner().
		Title("Logging in...").
		Action(func() {
			resp, err = cl.Auth().Login(context.Background(), challengeResponse)
		}).Run()
	if err != nil {
		return "", fmt.Errorf("failed to login: %w", err)
	}

	return resp.Token, nil
}
