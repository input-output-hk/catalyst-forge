package providers

import (
	"fmt"

	"github.com/input-output-hk/catalyst-forge/lib/project/project"
)

// parseConfig parses the configuration for the release.
func parseConfig(p *project.Project, release string, config any) error {
	return p.Raw().DecodePath(fmt.Sprintf("project.release.%s.config", release), &config)
}

// getPlatforms returns the platforms for the target.
func getPlatforms(p *project.Project, target string) []string {
	if _, ok := p.Blueprint.Project.CI.Targets[target]; ok {
		if len(p.Blueprint.Project.CI.Targets[target].Platforms) > 0 {
			return p.Blueprint.Project.CI.Targets[target].Platforms
		}
	}

	return nil
}
