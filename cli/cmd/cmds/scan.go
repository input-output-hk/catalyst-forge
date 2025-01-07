package cmds

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"

	"cuelang.org/go/cue"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/scan"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/utils"
	"golang.org/x/exp/maps"
)

type ScanCmd struct {
	Absolute  bool     `short:"a" help:"Output absolute paths."`
	Blueprint bool     `help:"Return the blueprint for each project."`
	Earthfile bool     `help:"Return the Earthfile targets for each project."`
	Filter    []string `short:"f" help:"Filter Earthfile targets by regular expression or blueprint results by path."`
	Pretty    bool     `help:"Pretty print JSON output."`
	RootPath  string   `kong:"arg,predictor=path" help:"Root path to scan for Earthfiles and their respective targets."`
}

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

	projects, err := scan.ScanProjects(rootPath, ctx.ProjectLoader, &ctx.FSWalker, ctx.Logger)
	if err != nil {
		return err
	}

	switch {
	case c.Blueprint && c.Earthfile:
		return fmt.Errorf("must specify one of --blueprint or --earthfile")
	case c.Blueprint && len(c.Filter) > 0:
		result := make(map[string]map[string]cue.Value)
		for path, project := range projects {
			for _, filter := range c.Filter {
				v := project.Raw().Get(filter)
				if v.Exists() {
					if _, ok := result[path]; !ok {
						result[path] = make(map[string]cue.Value)
					}
					result[path][filter] = v
				}
			}
		}

		utils.PrintJson(result, c.Pretty)
	case c.Blueprint:
		result := make(map[string]cue.Value)
		for path, project := range projects {
			result[path] = project.Raw().Value()
		}

		utils.PrintJson(result, c.Pretty)
	case c.Earthfile && len(c.Filter) > 0:
		result := make(map[string]map[string][]string)
		for _, filter := range c.Filter {
			filterExpr, err := regexp.Compile(filter)
			if err != nil {
				return err
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
	default:
		keys := maps.Keys(projects)
		sort.Strings(keys)
		utils.PrintJson(keys, c.Pretty)
	}

	return nil
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
