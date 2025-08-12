package cobra

import (
	"github.com/input-output-hk/catalyst-forge/cli/pkg/earthly"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	te "github.com/input-output-hk/catalyst-forge/lib/tools/earthly"
	"github.com/spf13/cobra"
)

// RunOptions holds the flags for the run command.
type RunOptions struct {
	Artifact   string
	Platform   []string
	Pretty     bool
	SkipOutput bool
	TargetArgs []string
}

// NewRunCommand creates the run command.
func NewRunCommand() *cobra.Command {
	opts := &RunOptions{}

	cmd := &cobra.Command{
		Use:   "run PATH [args...]",
		Short: "Run an Earthly target",
		Long:  `Execute Earthly targets with configuration and secret injection. The path should be in the format ./dir1+test.`,
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := run.MustFromContext(cmd.Context())
			opts.TargetArgs = args[1:]
			return runExecute(ctx, args[0], opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Artifact, "artifact", "a", "", "Dump all produced artifacts to the given path")
	cmd.Flags().StringSliceVarP(&opts.Platform, "platform", "p", nil, "Run the target with the given platform")
	cmd.Flags().BoolVar(&opts.Pretty, "pretty", false, "Pretty print JSON output")
	cmd.Flags().BoolVarP(&opts.SkipOutput, "skip-output", "s", false, "Skip outputting any images or artifacts")

	return cmd
}

// runExecute executes the run command logic.
func runExecute(ctx run.RunContext, path string, opts *RunOptions) error {
	ref, err := te.ParseEarthfileRef(path)
	if err != nil {
		return err
	}

	project, err := ctx.ProjectLoader.Load(ref.Path)
	if err != nil {
		return err
	}

	ctx.Logger.Info("Executing Earthly target", "project", project.Path, "target", ref.Target)
	runner := earthly.NewDefaultProjectRunner(ctx, &project)
	if err := runner.RunTarget(
		ref.Target,
		generateRunOpts(opts, ctx)...,
	); err != nil {
		return err
	}

	return nil
}

// generateRunOpts generates the options for the Earthly executor based on command flags.
func generateRunOpts(flags *RunOptions, ctx run.RunContext) []earthly.EarthlyExecutorOption {
	var opts []earthly.EarthlyExecutorOption

	if flags != nil {
		if flags.Artifact != "" {
			opts = append(opts, earthly.WithArtifact(flags.Artifact))
		}

		if ctx.CI {
			opts = append(opts, earthly.WithCI())
		}

		if flags.Platform != nil {
			opts = append(opts, earthly.WithPlatforms(flags.Platform...))
		}

		if len(flags.TargetArgs) > 0 && flags.TargetArgs[0] != "" {
			opts = append(opts, earthly.WithTargetArgs(flags.TargetArgs...))
		}

		if flags.SkipOutput {
			opts = append(opts, earthly.WithSkipOutput())
		}
	}

	return opts
}
