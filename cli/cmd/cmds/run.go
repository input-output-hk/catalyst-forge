package cmds

import (
	"github.com/input-output-hk/catalyst-forge/cli/pkg/earthly"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/lib/tools/earthfile"
)

type RunCmd struct {
	Artifact   string   `short:"a" help:"Dump all produced artifacts to the given path."`
	Path       string   `kong:"arg,predictor=path" help:"The path to the target to execute (i.e., ./dir1+test)."`
	Platform   []string `short:"p" help:"Run the target with the given platform."`
	Pretty     bool     `help:"Pretty print JSON output."`
	TargetArgs []string `arg:"" help:"Arguments to pass to the target." default:""`
}

func (c *RunCmd) Run(ctx run.RunContext) error {
	ref, err := earthfile.ParseEarthfileRef(c.Path)
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
		generateOpts(c, ctx)...,
	); err != nil {
		return err
	}

	return nil
}

// generateOpts generates the options for the Earthly executor based on command
// flags.
func generateOpts(flags *RunCmd, ctx run.RunContext) []earthly.EarthlyExecutorOption {
	var opts []earthly.EarthlyExecutorOption

	if flags != nil {
		if flags.Artifact != "" {
			opts = append(opts, earthly.WithArtifact(flags.Artifact))
		}

		if ctx.CI {
			opts = append(opts, earthly.WithCI())
		}

		// Users can explicitly set the platforms to use without being in CI mode.
		if flags.Platform != nil {
			opts = append(opts, earthly.WithPlatforms(flags.Platform...))
		}

		if len(flags.TargetArgs) > 0 && flags.TargetArgs[0] != "" {
			opts = append(opts, earthly.WithTargetArgs(flags.TargetArgs...))
		}
	}

	return opts
}
