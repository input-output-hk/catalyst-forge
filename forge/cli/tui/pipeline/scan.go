package pipeline

import (
	"regexp"

	"github.com/input-output-hk/catalyst-forge/forge/cli/internal/testutils"
	"github.com/input-output-hk/catalyst-forge/forge/cli/pkg/project"
	"github.com/input-output-hk/catalyst-forge/forge/cli/pkg/scan"
	"github.com/input-output-hk/catalyst-forge/tools/pkg/walker"
)

type runStatus string

const (
	runStatusIdle    runStatus = "idle"
	runStatusRunning runStatus = "running"
	runStatusFailed  runStatus = "failed"
	runStatusSuccess runStatus = "success"
)

// ProjectTarget represents a project and a target.
type ProjectTarget struct {
	project project.Project
	status  runStatus
	target  string
}

func scanProjects(
	startPath string,
	loader project.ProjectLoader,
	filters []string,
) ([][]*ProjectTarget, error) {
	w := walker.NewDefaultFSWalker(testutils.NewNoopLogger())
	var result [][]*ProjectTarget

	projects, err := scan.ScanProjects(startPath, loader, &w, testutils.NewNoopLogger())
	if err != nil {
		return nil, err
	}

	for _, filter := range filters {
		var pairs []*ProjectTarget

		filterExpr, err := regexp.Compile(filter)
		if err != nil {
			return nil, err
		}

		for _, project := range projects {
			if project.Earthfile != nil {
				targets := project.Earthfile.FilterTargets(func(target string) bool {
					return filterExpr.MatchString(target)
				})

				for _, target := range targets {
					pairs = append(pairs, &ProjectTarget{
						project: project,
						status:  runStatusIdle,
						target:  target,
					})
				}
			}
		}

		result = append(result, pairs)
	}

	return result, nil
}
