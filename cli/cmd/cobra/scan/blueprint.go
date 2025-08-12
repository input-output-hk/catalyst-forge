package scan

import (
	"cuelang.org/go/cue"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/utils"
	"github.com/input-output-hk/catalyst-forge/lib/project/project"
	"github.com/spf13/cobra"
)

// BlueprintOptions holds the flags for the blueprint scan command.
type BlueprintOptions struct {
	Absolute     bool
	Filter       []string
	FilterSource string
	Pretty       bool
}

// NewBlueprintCommand creates the scan blueprint subcommand.
func NewBlueprintCommand() *cobra.Command {
	opts := &BlueprintOptions{
		FilterSource: "path",
	}

	cmd := &cobra.Command{
		Use:   "blueprint ROOTPATH",
		Short: "Scan for projects by their blueprints",
		Long:  `Discover projects by scanning for blueprint.cue files in the filesystem.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := run.MustFromContext(cmd.Context())
			return blueprintExecute(ctx, args[0], opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.Absolute, "absolute", "a", false, "Output absolute paths")
	cmd.Flags().StringSliceVarP(&opts.Filter, "filter", "f", nil, "The filter expressions to use")
	cmd.Flags().StringVarP(&opts.FilterSource, "filter-source", "s", "path", "The source to filter by [path]")
	cmd.Flags().BoolVarP(&opts.Pretty, "pretty", "p", false, "Pretty print JSON output")

	return cmd
}

// blueprintExecute executes the scan blueprint command logic.
func blueprintExecute(ctx run.RunContext, rootPath string, opts *BlueprintOptions) error {
	projects, err := scanProjects(ctx, rootPath, opts.Absolute)
	if err != nil {
		return err
	}

	switch {
	case len(opts.Filter) > 0 && opts.FilterSource == "path":
		result := filterByPath(projects, opts.Filter)
		utils.PrintJson(result, opts.Pretty)
	default:
		result := make(map[string]cue.Value)
		for path, project := range projects {
			result[path] = project.Raw().Value()
		}
		utils.PrintJson(result, opts.Pretty)
	}

	return nil
}

// filterByPath filters the projects by blueprint paths using the given filters.
func filterByPath(projects map[string]project.Project, filters []string) map[string]map[string]cue.Value {
	result := make(map[string]map[string]cue.Value)
	for path, project := range projects {
		for _, filter := range filters {
			v := project.Raw().Get(filter)
			if v.Exists() {
				if _, ok := result[path]; !ok {
					result[path] = make(map[string]cue.Value)
				}
				result[path][filter] = v
			}
		}
	}
	return result
}
