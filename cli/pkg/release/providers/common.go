package providers

import (
	"fmt"
	"log/slog"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/providers/aws"
	"github.com/input-output-hk/catalyst-forge/lib/project/project"
)

// createECRRepoIfNotExists creates an ECR repository if it does not exist.
func createECRRepoIfNotExists(client aws.ECRClient, p *project.Project, registry string, logger *slog.Logger) error {
	name, err := aws.ExtractECRRepoName(registry)
	if err != nil {
		return fmt.Errorf("failed to extract ECR repository name: %w", err)
	}

	exists, err := client.ECRRepoExists(name)
	if err != nil {
		return fmt.Errorf("failed to check if ECR repository exists: %w", err)
	}

	if !exists {
		logger.Info("ECR repository does not exist, creating", "name", name)
		if err := client.CreateECRRepository(p, name); err != nil {
			return fmt.Errorf("failed to create ECR repository: %w", err)
		}
	}

	return nil
}

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
