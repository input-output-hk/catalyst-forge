package cobra

import (
	"fmt"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/release"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/lib/tools/fs"
	"github.com/spf13/cobra"
)

// ReleaseOptions holds the flags for the release command.
type ReleaseOptions struct {
	Force bool
}

// NewReleaseCommand creates the release command.
func NewReleaseCommand() *cobra.Command {
	opts := &ReleaseOptions{}

	cmd := &cobra.Command{
		Use:   "release PROJECT RELEASE",
		Short: "Release a project",
		Long:  `Execute project releases using configured providers with automatic CI mode activation.`,
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := run.MustFromContext(cmd.Context())
			return releaseExecute(ctx, args[0], args[1], opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.Force, "force", "f", false, "Force the release to run")

	return cmd
}

// releaseExecute executes the release command logic.
func releaseExecute(ctx run.RunContext, projectPath, releaseName string, opts *ReleaseOptions) error {
	exists, err := fs.Exists(projectPath)
	if err != nil {
		return fmt.Errorf("could not check if project exists: %w", err)
	} else if !exists {
		return fmt.Errorf("project does not exist: %s", projectPath)
	}

	project, err := ctx.ProjectLoader.Load(projectPath)
	if err != nil {
		return err
	}

	_, ok := project.Blueprint.Project.Release[releaseName]
	if !ok {
		return fmt.Errorf("unknown release: %s", releaseName)
	}

	// Always release in CI mode
	ctx.CI = true
	releasers := release.NewDefaultReleaserStore()
	releaser, err := releasers.GetReleaser(
		release.ReleaserType(releaseName),
		ctx,
		project,
		releaseName,
		opts.Force,
	)
	if err != nil {
		return fmt.Errorf("failed to initialize releaser: %w", err)
	}

	if err := releaser.Release(); err != nil {
		return fmt.Errorf("failed to release: %w", err)
	}

	return nil
}
