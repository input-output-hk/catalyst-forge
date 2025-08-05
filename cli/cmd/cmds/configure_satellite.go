package cmds

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/earthly/satellite"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/lib/tools/fs/billy"
	"github.com/input-output-hk/catalyst-forge/lib/tools/git"
	"github.com/input-output-hk/catalyst-forge/lib/tools/walker"
)

type ConfigureSatelliteCmd struct {
	Path string `short:"p" help:"Path to place the Earthly config and certificates."`
}

func (c *ConfigureSatelliteCmd) Run(ctx run.RunContext) error {
	fs := billy.NewBaseOsFS()
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %w", err)
	}

	ctx.Logger.Debug("Finding git root", "path", cwd)
	w := walker.NewCustomReverseFSWalker(fs, ctx.Logger)
	gitRoot, err := git.FindGitRoot(cwd, &w)
	if err != nil {
		return fmt.Errorf("failed to find git root: %w", err)
	}
	ctx.Logger.Debug("Git root found", "path", gitRoot)

	ctx.Logger.Debug("Loading project", "path", gitRoot)
	project, err := ctx.ProjectLoader.Load(gitRoot)
	if err != nil {
		return err
	}

	if c.Path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get user's home directory: %w", err)
		}

		c.Path = filepath.Join(home, ".earthly")
	}

	ctx.Logger.Info("Configuring satellite", "path", c.Path)
	satellite := satellite.NewEarthlySatellite(
		&project,
		c.Path,
		ctx.Logger,
		satellite.WithSecretStore(ctx.SecretStore),
		satellite.WithCI(ctx.CI),
	)
	if err := satellite.Configure(); err != nil {
		return fmt.Errorf("failed to configure satellite: %w", err)
	}

	return nil
}
