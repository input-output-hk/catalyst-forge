package common

import (
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/input-output-hk/catalyst-forge/lib/project/project"
	"github.com/input-output-hk/catalyst-forge/lib/providers/aws"
	s "github.com/input-output-hk/catalyst-forge/lib/schema"
)

var ErrConfigNotFound = fmt.Errorf("release config field not found")

// CreateECRRepoIfNotExists creates an ECR repository if it does not exist.
func CreateECRRepoIfNotExists(client aws.ECRClient, p *project.Project, registry string, logger *slog.Logger) error {
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
		if err := client.CreateECRRepository(name, p.Blueprint.Global.Repo.Name, p.Path); err != nil {
			return fmt.Errorf("failed to create ECR repository: %w", err)
		}
	}

	return nil
}

// IsECRRegistry checks if the registry is an ECR registry.
func IsECRRegistry(registry string) bool {
	return regexp.MustCompile(`^\d{12}\.dkr\.ecr\.[a-z0-9-]+\.amazonaws\.com`).MatchString(registry)
}

// ParseConfig parses the configuration for the release.
func ParseConfig(p *project.Project, release string, config any) error {
	err := p.Raw().DecodePath(fmt.Sprintf("project.release.%s.config", release), &config)

	if err != nil && strings.Contains(err.Error(), "not found") {
		return ErrConfigNotFound
	} else if err != nil {
		return err
	}

	return nil
}

// GetPlatforms returns the platforms for the target.
func GetPlatforms(p *project.Project, target string) []string {
	if s.HasProjectCiDefined(p.Blueprint) {
		if _, ok := p.Blueprint.Project.Ci.Targets[target]; ok {
			if len(p.Blueprint.Project.Ci.Targets[target].Platforms) > 0 {
				return p.Blueprint.Project.Ci.Targets[target].Platforms
			}
		}
	}

	return nil
}
