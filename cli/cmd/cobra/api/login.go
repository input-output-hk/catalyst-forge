package api

import (
	"context"
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/input-output-hk/catalyst-forge/cli/internal/state"
	"github.com/input-output-hk/catalyst-forge/cli/internal/ux"
	"github.com/input-output-hk/catalyst-forge/cli/internal/validator"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	authpkg "github.com/input-output-hk/catalyst-forge/lib/foundry/auth"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/client"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/client/auth"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/client/github"
	"github.com/spf13/cobra"
)

// LoginOptions holds the flags for the login command.
type LoginOptions struct {
	Email string
	Token string
	Type  string
}

// EmailForm represents the form for email input.
type EmailForm struct {
	Email string `form:"email"`
}

// NewLoginCommand creates the login command.
func NewLoginCommand() *cobra.Command {
	opts := &LoginOptions{
		Type: "foundry",
	}

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Login to the Foundry API",
		Long:  `Authenticate with the Foundry API using either GitHub token or Foundry credentials.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := run.MustFromContext(cmd.Context())
			cl, err := GetClient(cmd.Context())
			if err != nil {
				return err
			}
			return loginExecute(ctx, cl, opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Email, "email", "e", "", "The email of the user to login as")
	cmd.Flags().StringVar(&opts.Token, "token", "", "An existing JWT token to use for authentication")
	cmd.Flags().StringVarP(&opts.Type, "type", "t", "foundry", "The type of login to perform (github|foundry)")

	return cmd
}

// loginExecute executes the login command logic.
func loginExecute(ctx run.RunContext, cl client.Client, opts *LoginOptions) error {
	var jwt string
	var err error

	switch opts.Type {
	case "github":
		var resp *github.ValidateTokenResponse
		err = ux.NewSpinner().
			Title("Validating GitHub token...").
			Action(func() {
				resp, err = cl.Github().ValidateToken(context.Background(), &github.ValidateTokenRequest{
					Token: opts.Token,
				})
			}).Run()
		if err != nil {
			return fmt.Errorf("failed to login with github: %w", err)
		}
		jwt = resp.Token

	case "foundry":
		if opts.Token != "" {
			jwt = opts.Token
		} else {
			jwt, err = interactiveFoundryLogin(ctx, cl, opts)
			if err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("invalid login type: %s", opts.Type)
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

// interactiveFoundryLogin performs interactive Foundry authentication.
func interactiveFoundryLogin(ctx run.RunContext, cl client.Client, opts *LoginOptions) (string, error) {
	var form EmailForm
	var email string
	var err error

	// Use email from flag first, then config, then prompt
	if opts.Email != "" {
		email = opts.Email
	} else if ctx.Config.Email == "" {
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

	var challenge *auth.ChallengeResponse
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

	var loginRequest *authpkg.LoginRequest
	err = ux.NewSpinner().
		Title("Signing login challenge...").
		Action(func() {
			loginRequest, err = kp.SignChallenge(challenge.Token)
		}).Run()
	if err != nil {
		return "", fmt.Errorf("failed to sign challenge: %w", err)
	}

	var resp *auth.LoginResponse
	err = ux.NewSpinner().
		Title("Logging in...").
		Action(func() {
			resp, err = cl.Auth().Login(context.Background(), &auth.LoginRequest{
				Token:     loginRequest.Challenge,
				Signature: loginRequest.Signature,
			})
		}).Run()
	if err != nil {
		return "", fmt.Errorf("failed to login: %w", err)
	}

	return resp.Token, nil
}
