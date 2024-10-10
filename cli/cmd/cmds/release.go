package cmds

import (
	"fmt"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/release"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
)

type ReleaseCmd struct {
	Project string `arg:"" help:"Path to the project."`
	Release string `arg:"" help:"Name of the release."`
}

func (c *ReleaseCmd) Run(ctx run.RunContext) error {
	project, err := ctx.ProjectLoader.Load(c.Project)
	if err != nil {
		return err
	}

	if _, ok := project.Blueprint.Project.Release[c.Release]; !ok {
		return fmt.Errorf("unknown release: %s", c.Release)
	}

	// Always release in CI mode
	ctx.CI = true

	releasers := release.NewDefaultReleaserStore()
	releaser, err := releasers.GetReleaser(
		release.ReleaserTypeDocker,
		ctx,
		project,
		project.Blueprint.Project.Release[c.Release],
	)
	if err != nil {
		return fmt.Errorf("failed to initialize releaser: %w", err)
	}

	if err := releaser.Release(); err != nil {
		return fmt.Errorf("failed to release: %w", err)
	}

	return nil
}