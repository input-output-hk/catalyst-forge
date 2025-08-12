package module

import (
	"github.com/spf13/cobra"
)

// NewCommand creates the mod command with subcommands.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mod",
		Short: "Commands for working with deployment modules",
		Long:  `Manage deployment modules, templates, and GitOps deployments.`,
	}

	cmd.AddCommand(NewDeployCommand())
	cmd.AddCommand(NewDumpCommand())
	cmd.AddCommand(NewTemplateCommand())

	return cmd
}
