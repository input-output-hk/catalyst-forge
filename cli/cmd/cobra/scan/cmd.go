package scan

import (
	"github.com/spf13/cobra"
)

// NewCommand creates the scan command with subcommands.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "scan",
		Short: "Commands for scanning for projects",
		Long:  `Scan filesystem for projects, blueprints, and Earthfiles.`,
	}

	cmd.AddCommand(NewBlueprintCommand())
	cmd.AddCommand(NewEarthfileCommand())
	cmd.AddCommand(NewAllCommand())

	return cmd
}
