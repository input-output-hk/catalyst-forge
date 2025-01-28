package providers

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/google/go-github/v66/github"
	"github.com/input-output-hk/catalyst-forge/lib/project/project"
	"github.com/input-output-hk/catalyst-forge/lib/project/secrets"
)

// GithubProviderCreds is the struct that holds the credentials for the Github provider
type GithubProviderCreds struct {
	Token string
}

// GetGithubProviderCreds loads the Github provider credentials from the project.
func GetGithubProviderCreds(p *project.Project, logger *slog.Logger) (GithubProviderCreds, error) {
	secret := p.Blueprint.Global.CI.Providers.Github.Credentials
	if secret == nil {
		return GithubProviderCreds{}, fmt.Errorf("project does not have a Github provider configured")
	}

	m, err := secrets.GetSecretMap(secret, p.SecretStore, logger)
	if err != nil {
		return GithubProviderCreds{}, fmt.Errorf("could not get secret: %w", err)
	}

	creds, ok := m["token"]
	if !ok {
		return GithubProviderCreds{}, fmt.Errorf("github provider token is missing in secret")
	}

	return GithubProviderCreds{Token: creds}, nil
}

// NewGithubClient returns a new Github client.
// If a GITHUB_TOKEN environment variable is set, it will use that token.
// Otherwise, it will use the provider secret.
// If neither are set, it will create an anonymous client.
func NewGithubClient(p *project.Project, logger *slog.Logger) (*github.Client, error) {
	token, exists := os.LookupEnv("GITHUB_TOKEN")
	if exists {
		logger.Info("Creating Github client with environment token")
		return github.NewClient(nil).WithAuthToken(token), nil
	} else if p.Blueprint.Global.CI.Providers.Github.Credentials != nil {
		logger.Info("Creating Github client with provider secret")
		creds, err := GetGithubProviderCreds(p, logger)
		if err != nil {
			logger.Error("Failed to get Github provider credentials", "error", err)
		}

		return github.NewClient(nil).WithAuthToken(creds.Token), nil
	}

	logger.Info("Creating new anonymous Github client")
	return github.NewClient(nil), nil
}
