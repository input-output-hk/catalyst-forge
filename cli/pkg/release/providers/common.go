package providers

import (
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/providers/aws"
	"github.com/input-output-hk/catalyst-forge/lib/project/project"
)

var ErrConfigNotFound = fmt.Errorf("release config field not found")

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

// generateContainerName generates the container name for the project.
// If the name is not provided, the project name is used.
func generateContainerName(p *project.Project, name string, registry string) string {
	var n string
	if name == "" {
		n = p.Name
	} else {
		n = name
	}

	if isGHCRRegistry(registry) {
		return fmt.Sprintf("%s/%s", strings.TrimSuffix(registry, "/"), n)
	} else {
		var repo string
		if strings.Contains(p.Blueprint.Global.Repo.Name, "/") {
			repo = strings.Split(p.Blueprint.Global.Repo.Name, "/")[1]
		} else {
			repo = p.Blueprint.Global.Repo.Name
		}

		return fmt.Sprintf("%s/%s/%s", strings.TrimSuffix(registry, "/"), repo, n)
	}
}

// isECRRegistry checks if the registry is an ECR registry.
func isECRRegistry(registry string) bool {
	return regexp.MustCompile(`^\d{12}\.dkr\.ecr\.[a-z0-9-]+\.amazonaws\.com`).MatchString(registry)
}

// isGHCRRegistry checks if the registry is a GHCR registry.
func isGHCRRegistry(registry string) bool {
	return regexp.MustCompile(`^ghcr\.io/[a-zA-Z0-9](?:-?[a-zA-Z0-9])*$`).MatchString(registry)
}

// parseConfig parses the configuration for the release.
func parseConfig(p *project.Project, release string, config any) error {
	err := p.Raw().DecodePath(fmt.Sprintf("project.release.%s.config", release), &config)

	if err != nil && strings.Contains(err.Error(), "not found") {
		return ErrConfigNotFound
	} else if err != nil {
		return err
	}

	return nil
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
