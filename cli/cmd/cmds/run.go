package cmds

import (
	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/lib/tools/earthfile"
)

type RunCmd struct {
	Artifact   string   `short:"a" help:"Dump all produced artifacts to the given path."`
	Path       string   `arg:"" help:"The path to the target to execute (i.e., ./dir1+test)."`
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
	runner := run.NewProjectRunner(ctx, &project)
	result, err := runner.RunTarget(
		ref.Target,
		generateOpts(c, ctx)...,
	)
	if err != nil {
		return err
	}

	printJson(result, c.Pretty)
	return nil
}
