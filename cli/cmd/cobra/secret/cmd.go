package secret

import (
	"github.com/spf13/cobra"
)

// NewCommand creates the secret command with subcommands.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "secret",
		Short: "Manage secrets",
		Long:  `Get and set secrets using various providers like AWS Secrets Manager.`,
	}

	cmd.AddCommand(NewGetCommand())
	cmd.AddCommand(NewSetCommand())

	return cmd
}
