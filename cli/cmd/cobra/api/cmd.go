package api

import (
	"context"
	"fmt"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/utils"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/client"
	"github.com/spf13/cobra"
)

// NewCommand creates the api command with subcommands.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "api",
		Short: "Commands for working with the Foundry API",
		Long:  `Interact with Catalyst Foundry API for authentication, certificates, and user management.`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			ctx := run.MustFromContext(cmd.Context())

			cl, err := utils.NewAPIClient(ctx.RootProject, ctx)
			if err != nil {
				return fmt.Errorf("cannot create API client: %w", err)
			}

			// Store client in context for subcommands
			cmd.SetContext(WithClient(cmd.Context(), cl))
			return nil
		},
	}

	cmd.AddCommand(NewLoginCommand())
	cmd.AddCommand(NewRegisterCommand())

	return cmd
}

// WithClient stores the API client in the context.
func WithClient(ctx context.Context, cl client.Client) context.Context {
	return context.WithValue(ctx, clientKey{}, cl)
}

// GetClient retrieves the API client from the context.
func GetClient(ctx context.Context) (client.Client, error) {
	cl, ok := ctx.Value(clientKey{}).(client.Client)
	if !ok {
		return nil, fmt.Errorf("API client not found in context")
	}
	return cl, nil
}

type clientKey struct{}
