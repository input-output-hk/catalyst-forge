package git

import (
	"fmt"
	"log/slog"

	"github.com/input-output-hk/catalyst-forge/lib/providers/secrets"
	"github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/common"
)

// GitProviderCreds is the struct that holds the credentials for the Git provider
type GitProviderCreds struct {
	Token string
}

// GetGitProviderCreds loads the Git provider credentials from the given secret.
func GetGitProviderCreds(s *common.Secret, store *secrets.SecretStore, logger *slog.Logger) (GitProviderCreds, error) {
	m, err := secrets.GetSecretMap(s, store, logger)
	if err != nil {
		return GitProviderCreds{}, fmt.Errorf("could not get secret: %w", err)
	}

	creds, ok := m["token"]
	if !ok {
		return GitProviderCreds{}, fmt.Errorf("git provider token is missing in secret")
	}

	return GitProviderCreds{Token: creds}, nil
}
