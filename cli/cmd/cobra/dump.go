package cobra

import (
	"encoding/json"
	"fmt"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/spf13/cobra"
)

// DumpOptions holds the flags for the dump command.
type DumpOptions struct {
	Pretty bool
	Raw    bool
}

// NewDumpCommand creates the dump command.
func NewDumpCommand() *cobra.Command {
	opts := &DumpOptions{}

	cmd := &cobra.Command{
		Use:   "dump PROJECT",
		Short: "Dumps a project's blueprint to JSON",
		Long:  `Export project blueprints as JSON for inspection or processing.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := run.MustFromContext(cmd.Context())
			return dumpExecute(ctx, args[0], opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.Pretty, "pretty", "p", false, "Pretty print JSON output")
	cmd.Flags().BoolVar(&opts.Raw, "raw", false, "Output raw blueprint without processing")

	return cmd
}

// dumpExecute executes the dump command logic.
func dumpExecute(ctx run.RunContext, projectPath string, opts *DumpOptions) error {
	project, err := ctx.ProjectLoader.Load(projectPath)
	if err != nil {
		return fmt.Errorf("failed to load project: %w", err)
	}

	var output interface{}
	if opts.Raw {
		output = project.Raw()
	} else {
		output = project
	}

	var jsonBytes []byte
	if opts.Pretty {
		jsonBytes, err = json.MarshalIndent(output, "", "  ")
	} else {
		jsonBytes, err = json.Marshal(output)
	}

	if err != nil {
		return fmt.Errorf("failed to marshal project to JSON: %w", err)
	}

	fmt.Println(string(jsonBytes))
	return nil
}
