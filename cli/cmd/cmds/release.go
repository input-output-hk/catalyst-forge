package cmds

import (
	"fmt"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/release"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/lib/tools/fs"
)

type ReleaseCmd struct {
	Force   bool   `short:"f" help:"Force the release to run."`
	Project string `arg:"" help:"Path to the project."`
	Release string `arg:"" help:"Name of the release."`
}

func (c *ReleaseCmd) Run(ctx run.RunContext) error {
	exists, err := fs.Exists(c.Project)
	if err != nil {
		return fmt.Errorf("could not check if project exists: %w", err)
	} else if !exists {
		return fmt.Errorf("project does not exist: %s", c.Project)
	}

	project, err := ctx.ProjectLoader.Load(c.Project)
	if err != nil {
		return err
	}

	_, ok := project.Blueprint.Project.Release[c.Release]
	if !ok {
		return fmt.Errorf("unknown release: %s", c.Release)
	}

	// Always release in CI mode
	ctx.CI = true
	releasers := release.NewDefaultReleaserStore()
	releaser, err := releasers.GetReleaser(
		release.ReleaserType(c.Release),
		ctx,
		project,
		c.Release,
		c.Force,
	)
	if err != nil {
		return fmt.Errorf("failed to initialize releaser: %w", err)
	}

	if err := releaser.Release(); err != nil {
		return fmt.Errorf("failed to release: %w", err)
	}

	return nil
}
