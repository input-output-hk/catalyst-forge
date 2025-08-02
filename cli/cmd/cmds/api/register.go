package api

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/huh"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/foundry/api/client"
	"github.com/input-output-hk/catalyst-forge/foundry/api/client/users"
	"github.com/input-output-hk/catalyst-forge/foundry/api/pkg/auth"
)

type RegisterCmd struct {
	Email string `short:"e" help:"Email address to register"`
	Force bool   `short:"f" help:"Force registration even if the user already exists"`
}

type RegisterForm struct {
	Continue bool   `form:"continue"`
	Email    string `form:"email"`
}

func (c *RegisterCmd) Run(ctx run.RunContext, cl interface {
	Users() *users.UsersClient
	Keys() *users.KeysClient
}) error {
	var form RegisterForm

	manager := auth.NewAuthManager(auth.WithFilesystem(ctx.FS))
	stateDir, err := getStateDir(ctx)
	if err != nil {
		return err
	}

	formFlow := huh.NewForm(
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

	userExistFlow := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("User already exists").
				Description("This email address is already registered with Foundry. This key set will be registered to the existing user. Do you want to continue?").
				Value(&form.Continue),
		),
	)

	if c.Email != "" {
		form.Email = c.Email
	} else {

		if err := formFlow.Run(); err != nil {
			return fmt.Errorf("failed to run registration form: %w", err)
		}

		if !form.Continue {
			ctx.Logger.Info("Registration cancelled by user")
			return nil
		}
	}

	fmt.Println("Registering new user...")
	_, err = cl.Users().Register(context.Background(), &users.RegisterUserRequest{
		Email: form.Email,
	})

	// Check if the error is a conflict (user already exists)
	var apiErr *client.APIError
	if err != nil && errors.As(err, &apiErr) && apiErr.IsConflict() {
		if !c.Force {
			if err := userExistFlow.Run(); err != nil {
				return fmt.Errorf("failed to run user exist flow: %w", err)
			}

			if !form.Continue {
				ctx.Logger.Info("Registration cancelled by user")
				return nil
			}
		} else {
			fmt.Println("User already exists, registering key set...")
		}
	} else if err != nil {
		return fmt.Errorf("failed to register user: %w", err)
	} else {
		fmt.Printf("User %s registered successfully\n", form.Email)
	}

	fmt.Println("Generating key set...")
	keyset, err := manager.GenerateKeypair()
	if err != nil {
		return fmt.Errorf("failed to generate key set: %w", err)
	}

	fmt.Println("Saving key set...")
	if err := keyset.Save(stateDir); err != nil {
		return fmt.Errorf("failed to save key set: %w", err)
	}

	fmt.Println("Key set saved successfully")

	fmt.Println("Registering key set...")
	_, err = cl.Keys().Register(context.Background(), &users.RegisterUserKeyRequest{
		Email:     form.Email,
		Kid:       keyset.Kid(),
		PubKeyB64: keyset.EncodePublicKey(),
	})

	if err != nil {
		return fmt.Errorf("failed to register key set: %w", err)
	}

	fmt.Println("Updating config...")
	ctx.Config.Email = form.Email
	if err := ctx.Config.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Println("Registration complete! Please contact an administrator to activate your account.")
	return nil
}

// getStateDir gets the state directory for the CLI.
// It creates the directory if it doesn't exist.
func getStateDir(ctx run.RunContext) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	path := filepath.Join(home, ".local", "state", "forge")
	exists, err := ctx.FS.Exists(path)
	if err != nil {
		return "", fmt.Errorf("failed to check if state directory exists: %w", err)
	} else if !exists {
		ctx.Logger.Info("Creating state directory", "path", path)
		if err := ctx.FS.MkdirAll(path, 0755); err != nil {
			return "", fmt.Errorf("failed to create state directory: %s", err)
		}
	}

	return path, nil
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && (s[:len(substr)] == substr ||
			s[len(s)-len(substr):] == substr ||
			func() bool {
				for i := 0; i <= len(s)-len(substr); i++ {
					if s[i:i+len(substr)] == substr {
						return true
					}
				}
				return false
			}())))
}
