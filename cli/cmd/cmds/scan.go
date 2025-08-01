package cmds

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"

	"cuelang.org/go/cue"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/scan"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/utils"
	"github.com/input-output-hk/catalyst-forge/lib/project/project"
	"github.com/input-output-hk/catalyst-forge/lib/tools/fs"
	"golang.org/x/exp/maps"
)

type ScanCmd struct {
	Absolute     bool       `short:"a" help:"Output absolute paths."`
	Blueprint    bool       `help:"Return the blueprint for each project."`
	Earthfile    bool       `help:"Return the Earthfile targets for each project."`
	Filter       []string   `short:"f" help:"Filter Earthfile targets by regular expression or blueprint results by path."`
	FilterSource FilterType `short:"s" help:"The source to filter." enum:"blueprint,earthfile,targets" default:"targets"`
	Pretty       bool       `help:"Pretty print JSON output."`
	RootPath     string     `kong:"arg,predictor=path" help:"Root path to scan for Earthfiles and their respective targets."`
}

type FilterType string

const (
	FilterTypeBlueprint FilterType = "blueprint"
	FilterTypeEarthfile FilterType = "earthfile"
	FilterTypeTargets   FilterType = "targets"
)

func (c *ScanCmd) Run(ctx run.RunContext) error {
	var rootPath string
	if c.Absolute {
		var err error
		rootPath, err = filepath.Abs(c.RootPath)
		if err != nil {
			return fmt.Errorf("failed to get absolute path: %w", err)
		}
	} else {
		rootPath = c.RootPath
	}

	exists, err := fs.Exists(rootPath)
	if err != nil {
		return fmt.Errorf("could not check if root path exists: %w", err)
	} else if !exists {
		return fmt.Errorf("root path does not exist: %s", rootPath)
	}

	projects, err := scan.ScanProjects(rootPath, ctx.ProjectLoader, &ctx.FSWalker, ctx.Logger)
	if err != nil {
		return err
	}

	switch {
	case c.Blueprint && c.Earthfile:
		return fmt.Errorf("must specify one of --blueprint or --earthfile")
	case c.Blueprint && len(c.Filter) > 0:
		result := filterByBlueprint(projects, c.Filter)
		utils.PrintJson(result, c.Pretty)
	case c.Blueprint:
		result := make(map[string]cue.Value)
		for path, project := range projects {
			result[path] = project.Raw().Value()
		}

		utils.PrintJson(result, c.Pretty)
	case c.Earthfile && len(c.Filter) > 0:
		result, err := filterByTargets(ctx, projects, c.Filter)
		if err != nil {
			return err
		}

		if ctx.CI {
			enumerated := make(map[string][]string)
			for filter, targetMap := range result {
				enumerated[filter] = enumerate(targetMap)
				sort.Strings(enumerated[filter])
			}

			utils.PrintJson(enumerated, c.Pretty)
		} else {
			utils.PrintJson(result, c.Pretty)
		}
	case c.Earthfile:
		result := make(map[string][]string)
		for path, project := range projects {
			if project.Earthfile != nil {
				result[path] = project.Earthfile.Targets()
			}
		}

		if ctx.CI {
			enumerated := enumerate(result)
			sort.Strings(enumerated)
			utils.PrintJson(enumerated, c.Pretty)
		} else {
			utils.PrintJson(result, c.Pretty)
		}
	case c.FilterSource == FilterTypeEarthfile && len(c.Filter) > 0:
		result, err := filterByEarthfile(ctx, projects, c.Filter)
		if err != nil {
			return err
		}

		utils.PrintJson(result, c.Pretty)
	default:
		keys := maps.Keys(projects)
		sort.Strings(keys)
		utils.PrintJson(keys, c.Pretty)
	}

	return nil
}

// filterByBlueprint filters the projects by the blueprint using the given filters.
func filterByBlueprint(projects map[string]project.Project, filters []string) map[string]map[string]cue.Value {
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

// filterByEarthfile filters the projects by the Earthfile contents using the given filters.
func filterByEarthfile(ctx run.RunContext, projects map[string]project.Project, filters []string) (map[string][]string, error) {
	result := make(map[string][]string)
	for _, filter := range filters {
		filterExpr, err := regexp.Compile(filter)
		if err != nil {
			return nil, err
		}

		for _, project := range projects {
			if project.Earthfile != nil {
				path := filepath.Join(project.Path, "Earthfile")
				contents, err := os.ReadFile(path)
				if err != nil {
					return nil, fmt.Errorf("failed to read Earthfile: %w", err)
				}

				if filterExpr.Match(contents) {
					result[filter] = append(result[filter], path)
				}
			}
		}
	}

	return result, nil
}

// filterByTargets filters the projects by the targets using the given filters.
func filterByTargets(ctx run.RunContext, projects map[string]project.Project, filters []string) (map[string]map[string][]string, error) {
	result := make(map[string]map[string][]string)
	for _, filter := range filters {
		filterExpr, err := regexp.Compile(filter)
		if err != nil {
			return nil, err
		}

		for path, project := range projects {
			if project.Earthfile != nil {
				targets := project.Earthfile.FilterTargets(func(target string) bool {
					return filterExpr.MatchString(target)
				})

				if len(targets) > 0 {
					if _, ok := result[filter]; !ok {
						result[filter] = make(map[string][]string)
					}

					result[filter][path] = targets
				}

				ctx.Logger.Debug("Filtered Earthfile", "path", path, "targets", targets)
			}
		}
	}

	return result, nil
}

// enumerate enumerates the Earthfile+Target pairs from the target map.
func enumerate(data map[string][]string) []string {
	var result []string
	for path, targets := range data {
		for _, target := range targets {
			result = append(result, fmt.Sprintf("%s+%s", path, target))
		}
	}

	return result
}
