package deployment

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/project"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/secrets"
)

// GetGitToken loads the Git token from the project.
func GetGitToken(project *project.Project, store *secrets.SecretStore, logger *slog.Logger) (string, error) {
	secret := project.Blueprint.Global.CI.Providers.Git.Credentials
	if secret == nil {
		return "", fmt.Errorf("project does not have a Git provider configured")
	}

	strSecret, err := secrets.GetSecret(secret, store, logger)
	if err != nil {
		return "", fmt.Errorf("could not get secret: %w", err)
	}

	creds := struct {
		Token string `json:"token"`
	}{}
	if err := json.Unmarshal([]byte(strSecret), &creds); err != nil {
		return "", fmt.Errorf("could not unmarshal secret: %w", err)
	}

	if creds.Token == "" {
		return "", fmt.Errorf("git provider token is empty")
	}

	return creds.Token, nil
}
