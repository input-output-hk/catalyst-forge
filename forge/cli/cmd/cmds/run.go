package cmds

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/input-output-hk/catalyst-forge/forge/cli/pkg/earthly"
	"github.com/input-output-hk/catalyst-forge/forge/cli/pkg/executor"
	"github.com/input-output-hk/catalyst-forge/forge/cli/pkg/secrets"
)

type RunCmd struct {
	Artifact string `short:"a" help:"Dump all produced artifacts to the given path."`
	Local    bool   `short:"l" help:"Forces the target to run locally (ignores satellite)."`
	Path     string `arg:"" help:"The path to the target to execute (i.e., ./dir1+test)."`
	Pretty   bool   `help:"Pretty print JSON output."`
}

func (c *RunCmd) Run(logger *slog.Logger) error {
	if !strings.Contains(c.Path, "+") {
		return fmt.Errorf("invalid Earthfile+Target pair: %s", c.Path)
	}

	earthfileDir := strings.Split(c.Path, "+")[0]
	target := strings.Split(c.Path, "+")[1]

	config, err := loadBlueprint(earthfileDir, logger)
	if err != nil {
		return err
	}

	localExec := executor.NewLocalExecutor(
		logger,
		executor.WithRedirect(),
	)

	opts := generateOpts(target, c, &config)
	earthlyExec := earthly.NewEarthlyExecutor(
		earthfileDir,
		target,
		localExec,
		secrets.NewDefaultSecretStore(),
		logger,
		opts...,
	)

	logger.Info("Executing Earthly target", "earthfile", earthfileDir, "target", target)
	result, err := earthlyExec.Run()
	if err != nil {
		return err
	}

	printJson(result, c.Pretty)

	return nil
}
