package cobra

import (
	"fmt"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/spf13/cobra"
)

// NewValidateCommand creates the validate command.
func NewValidateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate PROJECT",
		Short: "Validates a project",
		Long:  `Validate project blueprints and configuration by attempting to load the project with the ProjectLoader.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := run.MustFromContext(cmd.Context())
			return validateExecute(ctx, args[0])
		},
	}

	return cmd
}

// validateExecute executes the validate command logic.
func validateExecute(ctx run.RunContext, projectPath string) error {
	ctx.Logger.Info("Validating project", "path", projectPath)

	_, err := ctx.ProjectLoader.Load(projectPath)
	if err != nil {
		return fmt.Errorf("project validation failed: %w", err)
	}

	ctx.Logger.Info("Project is valid", "path", projectPath)
	return nil
}
