package providers

import (
	"fmt"
	"log/slog"

	"github.com/input-output-hk/catalyst-forge/lib/project/project"
	"github.com/input-output-hk/catalyst-forge/lib/project/secrets"
)

// GitProviderCreds is the struct that holds the credentials for the Git provider
type GitProviderCreds struct {
	Token string
}

// GetGitProviderCreds loads the Git provider credentials from the project.
func GetGitProviderCreds(p *project.Project, logger *slog.Logger) (GitProviderCreds, error) {
	secret := p.Blueprint.Global.CI.Providers.Git.Credentials
	if secret == nil {
		return GitProviderCreds{}, fmt.Errorf("project does not have a Git provider configured")
	}

	m, err := secrets.GetSecretMap(secret, p.SecretStore, logger)
	if err != nil {
		return GitProviderCreds{}, fmt.Errorf("could not get secret: %w", err)
	}

	creds, ok := m["token"]
	if !ok {
		return GitProviderCreds{}, fmt.Errorf("git provider token is missing in secret")
	}

	return GitProviderCreds{Token: creds}, nil
}
