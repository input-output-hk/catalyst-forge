package cobra

import (
	"github.com/input-output-hk/catalyst-forge/cli/pkg/earthly"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/cli/tui/ci"
	"github.com/spf13/cobra"
)

// CIOptions holds the flags for the CI command.
type CIOptions struct {
	Artifact string
	Platform []string
}

// NewCICommand creates the CI command.
func NewCICommand() *cobra.Command {
	opts := &CIOptions{}

	cmd := &cobra.Command{
		Use:   "ci [PATH]",
		Short: "Simulate a CI run",
		Long:  `Simulate a CI environment locally with interactive terminal interface for real-time execution monitoring.`,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := run.MustFromContext(cmd.Context())

			path := ""
			if len(args) > 0 {
				path = args[0]
			}

			return ciExecute(ctx, path, opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Artifact, "artifact", "a", "", "Dump all produced artifacts to the given path")
	cmd.Flags().StringSliceVarP(&opts.Platform, "platform", "p", nil, "Run the target with the given platform")

	return cmd
}

// ciExecute executes the CI command logic.
func ciExecute(ctx run.RunContext, path string, opts *CIOptions) error {
	// Convert to earthly options for the CI TUI
	earthlyOpts := generateCIOpts(opts, ctx)
	return ci.Run(path, ctx, earthlyOpts...)
}

// generateCIOpts generates the options for the Earthly executor based on CI command flags.
func generateCIOpts(flags *CIOptions, ctx run.RunContext) []earthly.EarthlyExecutorOption {
	var opts []earthly.EarthlyExecutorOption

	if flags != nil {
		if flags.Artifact != "" {
			opts = append(opts, earthly.WithArtifact(flags.Artifact))
		}

		// CI mode is always enabled for CI command
		opts = append(opts, earthly.WithCI())

		if flags.Platform != nil {
			opts = append(opts, earthly.WithPlatforms(flags.Platform...))
		}
	}

	return opts
}
