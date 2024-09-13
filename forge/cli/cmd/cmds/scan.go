package cmds

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"regexp"
	"sort"

	"cuelang.org/go/cue"
	"github.com/input-output-hk/catalyst-forge/forge/cli/pkg/project"
	"github.com/input-output-hk/catalyst-forge/forge/cli/pkg/scan"
	"github.com/input-output-hk/catalyst-forge/tools/pkg/walker"
	"golang.org/x/exp/maps"
)

type ScanCmd struct {
	Absolute  bool     `short:"a" help:"Output absolute paths."`
	Blueprint bool     `help:"Return the blueprint for each project."`
	CI        bool     `help:"Run in CI mode."`
	Earthfile bool     `help:"Return the Earthfile targets for each project."`
	Filter    []string `short:"f" help:"Filter Earthfile targets by regular expression or blueprint results by path."`
	Pretty    bool     `help:"Pretty print JSON output."`
	RootPath  string   `arg:"" help:"Root path to scan for Earthfiles and their respective targets."`
}

func (c *ScanCmd) Run(logger *slog.Logger) error {
	walker := walker.NewDefaultFSWalker(logger)
	loader := project.NewDefaultProjectLoader(loadRuntimes(logger), logger)

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

	projects, err := scan.ScanProjects(rootPath, &loader, &walker, logger)
	if err != nil {
		return err
	}

	switch {
	case c.Blueprint && c.Earthfile:
		return fmt.Errorf("must specify one of --blueprint or --earthfile")
	case c.Blueprint && len(c.Filter) > 0:
		result := make(map[string]map[string]cue.Value)
		for path, project := range projects {
			result[path] = make(map[string]cue.Value)
			for _, filter := range c.Filter {
				v := project.Raw().Get(filter)
				if v.Exists() {
					result[path][filter] = v
				}
			}
		}

		printJson(result, c.Pretty)
	case c.Blueprint:
		result := make(map[string]cue.Value)
		for path, project := range projects {
			result[path] = project.Raw().Value()
		}

		printJson(result, c.Pretty)
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

					logger.Debug("Filtered Earthfile", "path", path, "targets", targets)
				}
			}
		}

		if c.CI {
			enumerated := make(map[string][]string)
			for filter, targetMap := range result {
				enumerated[filter] = enumerate(targetMap)
				sort.Strings(enumerated[filter])
			}

			printJson(enumerated, c.Pretty)
		} else {
			printJson(result, c.Pretty)
		}
	case c.Earthfile:
		result := make(map[string][]string)
		for path, project := range projects {
			if project.Earthfile != nil {
				result[path] = project.Earthfile.Targets()
			}
		}

		if c.CI {
			enumerated := enumerate(result)
			sort.Strings(enumerated)
			printJson(enumerated, c.Pretty)
		} else {
			printJson(result, c.Pretty)
		}
	default:
		keys := maps.Keys(projects)
		sort.Strings(keys)
		printJson(keys, c.Pretty)
	}

	return nil
}
