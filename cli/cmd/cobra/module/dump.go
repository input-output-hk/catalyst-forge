package module

import (
	"fmt"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/lib/deployment"
	"github.com/input-output-hk/catalyst-forge/lib/tools/fs"
	"github.com/spf13/cobra"
)

// NewDumpCommand creates the module dump subcommand.
func NewDumpCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dump PROJECT",
		Short: "Dumps a project's deployment modules",
		Long:  `Export project deployment modules in their processed form for inspection.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := run.MustFromContext(cmd.Context())
			return dumpExecute(ctx, args[0])
		},
	}

	return cmd
}

// dumpExecute executes the module dump command logic.
func dumpExecute(ctx run.RunContext, projectPath string) error {
	exists, err := fs.Exists(projectPath)
	if err != nil {
		return fmt.Errorf("could not check if project exists: %w", err)
	} else if !exists {
		return fmt.Errorf("project does not exist: %s", projectPath)
	}

	project, err := ctx.ProjectLoader.Load(projectPath)
	if err != nil {
		return fmt.Errorf("could not load project: %w", err)
	}

	bundle := deployment.NewModuleBundle(&project)
	result, err := bundle.Dump()
	if err != nil {
		return fmt.Errorf("failed to dump deployment modules: %w", err)
	}

	fmt.Print(string(result))
	return nil
}
