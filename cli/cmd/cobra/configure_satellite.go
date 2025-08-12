package cobra

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/earthly/satellite"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/lib/tools/fs/billy"
	"github.com/input-output-hk/catalyst-forge/lib/tools/git"
	"github.com/input-output-hk/catalyst-forge/lib/tools/walker"
	"github.com/spf13/cobra"
)

// ConfigureSatelliteOptions holds the flags for the configure-satellite command.
type ConfigureSatelliteOptions struct {
	Path string
}

// NewConfigureSatelliteCommand creates the configure-satellite command.
func NewConfigureSatelliteCommand() *cobra.Command {
	opts := &ConfigureSatelliteOptions{}

	cmd := &cobra.Command{
		Use:   "configure-satellite",
		Short: "Configure the local system to use a remote Earthly Satellite",
		Long:  `Configure Earthly satellite connection with proper certificates and configuration for remote builds.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := run.MustFromContext(cmd.Context())
			return configureSatelliteExecute(ctx, opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Path, "path", "p", "", "Path to place the Earthly config and certificates")

	return cmd
}

// configureSatelliteExecute executes the configure-satellite command logic.
func configureSatelliteExecute(ctx run.RunContext, opts *ConfigureSatelliteOptions) error {
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

	if opts.Path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get user's home directory: %w", err)
		}

		opts.Path = filepath.Join(home, ".earthly")
	}

	ctx.Logger.Info("Configuring satellite", "path", opts.Path)
	sat := satellite.NewEarthlySatellite(
		&project,
		opts.Path,
		ctx.Logger,
		satellite.WithSecretStore(ctx.SecretStore),
		satellite.WithCI(ctx.CI),
	)
	if err := sat.Configure(); err != nil {
		return fmt.Errorf("failed to configure satellite: %w", err)
	}

	return nil
}
