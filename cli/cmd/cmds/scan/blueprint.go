package scan

import (
	"cuelang.org/go/cue"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/utils"
	"github.com/input-output-hk/catalyst-forge/lib/project/project"
)

type BlueprintCmd struct {
	Absolute     bool                `short:"a" help:"Output absolute paths."`
	Filter       []string            `short:"f" help:"The filter expressions to use."`
	FilterSource BlueprintFilterType `short:"s" help:"The source to filter by [path]." default:"path"`
	Pretty       bool                `short:"p" help:"Pretty print JSON output."`
	RootPath     string              `kong:"arg,predictor=path" help:"Root path to scan for projects."`
}

type BlueprintFilterType string

const (
	FilterTypePath BlueprintFilterType = "path"
)

func (c *BlueprintCmd) Run(ctx run.RunContext) error {
	projects, err := scanProjects(ctx, c.RootPath, c.Absolute)
	if err != nil {
		return err
	}

	switch {
	case len(c.Filter) > 0 && c.FilterSource == FilterTypePath:
		result := filterByPath(projects, c.Filter)
		utils.PrintJson(result, c.Pretty)
	default:
		result := make(map[string]cue.Value)
		for path, project := range projects {
			result[path] = project.Raw().Value()
		}

		utils.PrintJson(result, c.Pretty)
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
