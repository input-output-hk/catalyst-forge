package secrets

import (
	"fmt"
	"log/slog"

	"github.com/input-output-hk/catalyst-forge/lib/blueprint/schema"
)

// GetSecret returns the secret value for the given secret.
func GetSecret(s *schema.Secret, store *SecretStore, logger *slog.Logger) (string, error) {
	provider, err := store.NewClient(logger, Provider(s.Provider))
	if err != nil {
		return "", fmt.Errorf("failed to get secret provider: %w", err)
	}

	return provider.Get(s.Path)
}
