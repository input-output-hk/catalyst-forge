package cmds

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"regexp"
	"sort"

	"github.com/input-output-hk/catalyst-forge/forge/cli/pkg/earthfile"
	"github.com/input-output-hk/catalyst-forge/forge/cli/pkg/walker"
)

type ScanCmd struct {
	Absolute  bool     `short:"a" help:"Output absolute paths."`
	Enumerate bool     `short:"e" help:"Enumerate results into Earthfile+Target pairs."`
	Filter    []string `short:"f" help:"Filter discovered Earthfiles by target name using a regular expression."`
	Pretty    bool     `help:"Pretty print JSON output."`
	RootPath  string   `arg:"" help:"Root path to scan for Earthfiles and their respective targets."`
}

func (c *ScanCmd) Run(logger *slog.Logger) error {
	walker := walker.NewFilesystemWalker(logger)

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

	earthfiles, err := earthfile.ScanEarthfiles(rootPath, &walker, logger)
	if err != nil {
		return err
	}

	if len(c.Filter) > 0 {
		result := make(map[string]map[string][]string)
		for _, filter := range c.Filter {
			filterExpr, err := regexp.Compile(filter)
			if err != nil {
				return err
			}

			for path, earthfile := range earthfiles {
				targets := earthfile.FilterTargets(func(target string) bool {
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

		if c.Enumerate {
			enumerated := make(map[string][]string)
			for filter, targetMap := range result {
				enumerated[filter] = enumerate(targetMap)
				sort.Strings(enumerated[filter]) // Sort to provide deterministic output
			}

			printJson(enumerated, c.Pretty)
		} else {
			printJson(result, c.Pretty)
		}
	} else {
		targetMap := make(map[string][]string)
		for path, earthfile := range earthfiles {
			targetMap[path] = earthfile.Targets()
		}

		if c.Enumerate {
			enumerated := enumerate(targetMap)
			sort.Strings(enumerated) // Sort to provide deterministic output
			printJson(enumerated, c.Pretty)
		} else {
			printJson(targetMap, c.Pretty)
		}
	}

	return nil
}
