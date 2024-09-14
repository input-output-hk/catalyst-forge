package cmds

import (
	"log/slog"

	"github.com/input-output-hk/catalyst-forge/forge/cli/pkg/earthfile"
	"github.com/input-output-hk/catalyst-forge/forge/cli/pkg/executor"
	"github.com/input-output-hk/catalyst-forge/forge/cli/pkg/secrets"
)

type RunCmd struct {
	Artifact   string   `short:"a" help:"Dump all produced artifacts to the given path."`
	CI         bool     `help:"Run the target in CI mode."`
	Local      bool     `short:"l" help:"Forces the target to run locally (ignores satellite)."`
	Path       string   `arg:"" help:"The path to the target to execute (i.e., ./dir1+test)."`
	Platform   []string `short:"p" help:"Run the target with the given platform."`
	Pretty     bool     `help:"Pretty print JSON output."`
	TargetArgs []string `arg:"" help:"Arguments to pass to the target." default:""`
}

func (c *RunCmd) Run(logger *slog.Logger) error {
	ref, err := earthfile.ParseEarthfileRef(c.Path)
	if err != nil {
		return err
	}

	project, err := loadProject(ref.Path, logger)
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
		c.CI,
		c.Local,
		localExec,
		secrets.NewDefaultSecretStore(),
		generateOpts(c)...,
	)
	if err != nil {
		return err
	}

	printJson(result, c.Pretty)

	return nil
}
