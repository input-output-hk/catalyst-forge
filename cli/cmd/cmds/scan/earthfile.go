package scan

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/utils"
	"github.com/input-output-hk/catalyst-forge/lib/project/project"
)

type EarthfileCmd struct {
	Absolute     bool       `short:"a" help:"Output absolute paths."`
	Combine      bool       `short:"c" help:"Combine all filter results."`
	Enumerate    bool       `short:"e" help:"Enumerate the Earthfile+Target pairs."`
	Filter       []string   `short:"f" help:"The filter expressions to use."`
	FilterSource FilterType `short:"s" help:"The source to filter by [earthfile | targets]." default:"targets"`
	Pretty       bool       `short:"p" help:"Pretty print JSON output."`
	RootPath     string     `kong:"arg,predictor=path" help:"Root path to scan for Earthfiles and their respective targets."`
	Tag          []string   `short:"t" help:"The tags to filter by (only used when filtering by targets)."`
}

type FilterType string

const (
	FilterTypeTargets   FilterType = "targets"
	FilterTypeEarthfile FilterType = "earthfile"
)

func (c *EarthfileCmd) Run(ctx run.RunContext) error {
	projects, err := scanProjects(ctx, c.RootPath, c.Absolute)
	if err != nil {
		return err
	}

	switch {
	case len(c.Filter) > 0 && c.FilterSource == FilterTypeTargets:
		if len(c.Filter) == 0 {
			return fmt.Errorf("no filters provided")
		}

		result, err := filterByTargets(ctx, projects, c.Filter)
		if err != nil {
			return err
		}

		if len(c.Tag) > 0 {
			result = filterByTags(projects, result, c.Tag)
		}

		if c.Enumerate {
			enumerated := make(map[string][]string)
			for filter, targetMap := range result {
				enumerated[filter] = enumerate(targetMap)
				sort.Strings(enumerated[filter])
			}

			if c.Combine {
				var combined []string
				for _, targets := range enumerated {
					combined = append(combined, targets...)
				}

				sort.Strings(deduplicate(combined))
				utils.PrintJson(combined, c.Pretty)
			} else {
				utils.PrintJson(enumerated, c.Pretty)
			}
		} else {
			utils.PrintJson(result, c.Pretty)
		}
	case len(c.Filter) > 0 && c.FilterSource == FilterTypeEarthfile:
		if len(c.Filter) == 0 {
			return fmt.Errorf("no filters provided")
		}

		result, err := filterByEarthfile(projects, c.Filter)
		if err != nil {
			return err
		}

		if c.Combine {
			var combined []string
			for _, targets := range result {
				combined = append(combined, targets...)
			}

			sort.Strings(deduplicate(combined))
			utils.PrintJson(combined, c.Pretty)
		} else {
			utils.PrintJson(result, c.Pretty)
		}
	default:
		result := make(map[string][]string)
		for path, project := range projects {
			if project.Earthfile != nil {
				result[path] = project.Earthfile.Targets()
			}
		}

		if c.Enumerate {
			enumerated := enumerate(result)
			sort.Strings(enumerated)
			utils.PrintJson(enumerated, c.Pretty)
		} else {
			utils.PrintJson(result, c.Pretty)
		}
	}

	return nil
}

// filterByEarthfile filters the projects by the Earthfile contents using the given filters.
func filterByEarthfile(projects map[string]project.Project, filters []string) (map[string][]string, error) {
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

// filterByTags filters the scan results by the given tags.
func filterByTags(projects map[string]project.Project, input map[string]map[string][]string, tags []string) map[string]map[string][]string {
	result := make(map[string]map[string][]string)
	for filter, targetMap := range input {
		if _, ok := result[filter]; !ok {
			result[filter] = make(map[string][]string)
		}

		for path, targets := range targetMap {
			for _, target := range targets {
				project := projects[path]
				if project.Blueprint.Project != nil &&
					project.Blueprint.Project.Ci != nil &&
					project.Blueprint.Project.Ci.Targets != nil {
					if targetConfig, ok := project.Blueprint.Project.Ci.Targets[target]; ok {
						for _, tag := range targetConfig.Tags {
							if slices.Contains(tags, tag) {
								result[filter][path] = append(result[filter][path], target)
							}
						}
					}
				}
			}
		}
	}

	return result
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

// deduplicate removes duplicate strings from a slice while preserving order.
func deduplicate(slice []string) []string {
	seen := make(map[string]bool)
	var result []string

	for _, item := range slice {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}

	return result
}
