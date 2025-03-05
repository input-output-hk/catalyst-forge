package providers

import (
	"fmt"
	"log/slog"

	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/input-output-hk/catalyst-forge/lib/project/project"
	"github.com/input-output-hk/catalyst-forge/lib/project/secrets"
)

// GitProviderCreds is the struct that holds the credentials for the Git provider
type GitProviderCreds struct {
	Token string
}

// GetGitProviderCreds loads the Git provider credentials from the project.
func GetGitProviderCreds(p *project.Project, logger *slog.Logger) (GitProviderCreds, error) {
	secret := p.Blueprint.Global.Ci.Providers.Git.Credentials
	m, err := secrets.GetSecretMap(&secret, &p.SecretStore, logger)
	if err != nil {
		return GitProviderCreds{}, fmt.Errorf("could not get secret: %w", err)
	}

	creds, ok := m["token"]
	if !ok {
		return GitProviderCreds{}, fmt.Errorf("git provider token is missing in secret")
	}

	return GitProviderCreds{Token: creds}, nil
}

// LoadGitProviderCreds loads the Git provider credentials into the project's
// repository.
func LoadGitProviderCreds(p *project.Project, logger *slog.Logger) error {
	creds, err := GetGitProviderCreds(p, logger)
	if err != nil {
		return fmt.Errorf("could not get git provider credentials: %w", err)
	}

	p.Repo.SetAuth(&http.BasicAuth{
		Username: "forge",
		Password: creds.Token,
	})

	return nil
}
