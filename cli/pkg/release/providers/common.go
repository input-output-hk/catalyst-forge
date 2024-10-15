package providers

import "github.com/input-output-hk/catalyst-forge/lib/project/project"

// getPlatforms returns the platforms for the target.
func getPlatforms(p *project.Project, target string) []string {
	if _, ok := p.Blueprint.Project.CI.Targets[target]; ok {
		if len(p.Blueprint.Project.CI.Targets[target].Platforms) > 0 {
			return p.Blueprint.Project.CI.Targets[target].Platforms
		}
	}

	return nil
}
