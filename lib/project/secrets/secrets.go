package secrets

import (
	"encoding/json"
	"fmt"
	"log/slog"

	sc "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/common"
)

// GetSecret returns the secret value for the given secret.
func GetSecret(s *sc.Secret, store *SecretStore, logger *slog.Logger) (string, error) {
	provider, err := store.NewClient(logger, Provider(s.Provider))
	if err != nil {
		return "", fmt.Errorf("failed to get secret provider: %w", err)
	}

	return provider.Get(s.Path)
}

// GetSecretMap returns the secret value for the given secret as a map.
func GetSecretMap(s *sc.Secret, store *SecretStore, logger *slog.Logger) (map[string]string, error) {
	provider, err := store.NewClient(logger, Provider(s.Provider))
	if err != nil {
		return nil, fmt.Errorf("failed to get secret provider: %w", err)
	}

	secret, err := provider.Get(s.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to get secret: %w", err)
	}

	var secretMap map[string]string
	if err := json.Unmarshal([]byte(secret), &secretMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal secret: %w", err)
	}

	var finalMap map[string]string
	if s.Maps != nil {
		finalMap := make(map[string]string)
		for k, v := range s.Maps {
			if _, ok := secretMap[k]; !ok {
				return nil, fmt.Errorf("secret key not found in secret values: %s", k)
			}

			finalMap[k] = secretMap[v]
		}
	} else {
		finalMap = secretMap
	}

	return finalMap, nil
}
