package github

import (
	"fmt"
	"log/slog"

	"github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/common"
	"github.com/input-output-hk/catalyst-forge/lib/secrets"
)

// GithubProviderCreds is the struct that holds the credentials for the Github provider
type GithubProviderCreds struct {
	Token string
}

// getGithubProviderCreds loads the Github provider credentials from the project.
func getGithubProviderCreds(s *common.Secret, ss *secrets.SecretStore, logger *slog.Logger) (GithubProviderCreds, error) {
	m, err := secrets.GetSecretMap(s, ss, logger)
	if err != nil {
		return GithubProviderCreds{}, fmt.Errorf("could not get secret: %w", err)
	}

	creds, ok := m["token"]
	if !ok {
		return GithubProviderCreds{}, fmt.Errorf("github provider token is missing in secret")
	}

	return GithubProviderCreds{Token: creds}, nil
}
