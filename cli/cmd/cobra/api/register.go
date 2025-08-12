package api

import (
	"context"
	"errors"
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/input-output-hk/catalyst-forge/cli/internal/state"
	"github.com/input-output-hk/catalyst-forge/cli/internal/ux"
	"github.com/input-output-hk/catalyst-forge/cli/internal/validator"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/auth"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/client"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/client/users"
	"github.com/spf13/cobra"
)

// RegisterOptions holds the flags for the register command.
type RegisterOptions struct {
	Email string
	Force bool
}

// RegisterForm represents the form for registration input.
type RegisterForm struct {
	Continue bool   `form:"continue"`
	Email    string `form:"email"`
}

// NewRegisterCommand creates the register command.
func NewRegisterCommand() *cobra.Command {
	opts := &RegisterOptions{}

	cmd := &cobra.Command{
		Use:   "register",
		Short: "Register a new user with the Foundry API",
		Long:  `Register a new user account and generate cryptographic keys for API authentication.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := run.MustFromContext(cmd.Context())
			cl, err := GetClient(cmd.Context())
			if err != nil {
				return err
			}
			return registerExecute(ctx, cl, opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Email, "email", "e", "", "Email address to register")
	cmd.Flags().BoolVarP(&opts.Force, "force", "f", false, "Force registration even if the user already exists")

	return cmd
}

// registerExecute executes the register command logic.
func registerExecute(ctx run.RunContext, cl client.Client, opts *RegisterOptions) error {
	var form RegisterForm

	manager := auth.NewAuthManager(auth.WithFilesystem(ctx.FS))
	stateDir, err := state.GetDir(ctx)
	if err != nil {
		return err
	}

	registrationForm := ux.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Register with Foundry API").
				Description("This will generate a new key set on this machine and register it in the API, do you want to continue?").
				Value(&form.Continue),
		),
		huh.NewGroup(
			huh.NewInput().
				Title("Work email").
				Description("Enter your work email address").
				Placeholder("user@company.com").
				Value(&form.Email).
				Validate(validator.Email),
		),
	)

	userExistsConfirmation := ux.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("User already exists").
				Description("This email address is already registered with Foundry. This key set will be registered to the existing user. Do you want to continue?").
				Value(&form.Continue),
		),
	)

	if opts.Email != "" {
		form.Email = opts.Email
	} else {
		if err := registrationForm.Run(); err != nil {
			return fmt.Errorf("failed to run registration form: %w", err)
		}

		if !form.Continue {
			ux.Info("Registration cancelled by user")
			return nil
		}
	}

	err = ux.NewSpinner().
		Title("Registering new user...").
		Action(func() {
			_, err = cl.Users().Register(context.Background(), &users.RegisterUserRequest{
				Email: form.Email,
			})
		}).Run()

	// Check if the error is a conflict (user already exists)
	var apiErr *client.APIError
	if err != nil && errors.As(err, &apiErr) && apiErr.IsConflict() {
		if !opts.Force {
			if err := userExistsConfirmation.Run(); err != nil {
				return fmt.Errorf("failed to run user exist flow: %w", err)
			}
			if !form.Continue {
				ux.Info("Registration cancelled by user")
				return nil
			}
		} else {
			ux.Info("User already exists, registering key set...")
		}
	} else if err != nil {
		return fmt.Errorf("failed to register user: %w", err)
	} else {
		ux.Successfln("User %s registered successfully", form.Email)
	}

	var keyset *auth.KeyPair
	err = ux.NewSpinner().
		Title("Generating key set...").
		Action(func() {
			keyset, err = manager.GenerateKeypair()
		}).Run()
	if err != nil {
		return fmt.Errorf("failed to generate key set: %w", err)
	}

	err = ux.NewSpinner().
		Title("Saving key set...").
		Action(func() {
			if err = keyset.Save(stateDir); err != nil {
				err = fmt.Errorf("failed to save key set: %w", err)
			}
		}).Run()
	if err != nil {
		return err
	}
	ux.Success("Key set saved successfully")

	err = ux.NewSpinner().
		Title("Registering key set...").
		Action(func() {
			_, err = cl.Keys().Register(context.Background(), &users.RegisterUserKeyRequest{
				Email:     form.Email,
				Kid:       keyset.Kid(),
				PubKeyB64: keyset.EncodePublicKey(),
			})
		}).Run()

	if err != nil {
		return fmt.Errorf("failed to register key set: %w", err)
	}

	err = ux.NewSpinner().
		Title("Updating config...").
		Action(func() {
			ctx.Config.Email = form.Email
			if err = ctx.Config.Save(); err != nil {
				err = fmt.Errorf("failed to save config: %w", err)
			}
		}).Run()

	if err != nil {
		return err
	}

	ux.Success("Registration complete! Please contact an administrator to activate your account.")
	return nil
}
