package cmds

import (
	"log/slog"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/earthfile"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/executor"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/secrets"
)

type RunCmd struct {
	Artifact   string   `short:"a" help:"Dump all produced artifacts to the given path."`
	Path       string   `arg:"" help:"The path to the target to execute (i.e., ./dir1+test)."`
	Platform   []string `short:"p" help:"Run the target with the given platform."`
	Pretty     bool     `help:"Pretty print JSON output."`
	TargetArgs []string `arg:"" help:"Arguments to pass to the target." default:""`
}

func (c *RunCmd) Run(logger *slog.Logger, global GlobalArgs) error {
	ref, err := earthfile.ParseEarthfileRef(c.Path)
	if err != nil {
		return err
	}

	project, err := loadProject(global, ref.Path, logger)
	if err != nil {
		return err
	}

	logger.Info("Executing Earthly target", "project", project.Path, "target", ref.Target)
	localExec := executor.NewLocalExecutor(
		logger,
		executor.WithRedirect(),
	)
	result, err := project.RunTarget(
		ref.Target,
		localExec,
		secrets.NewDefaultSecretStore(),
		generateOpts(c, &global)...,
	)
	if err != nil {
		return err
	}

	printJson(result, c.Pretty)
	return nil
}
