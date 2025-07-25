package test

import (
	"encoding/json"
	"log/slog"

	"github.com/input-output-hk/catalyst-forge/lib/secrets"
	sm "github.com/input-output-hk/catalyst-forge/lib/secrets/mocks"
)

func NewMockSecretStore(result map[string]string) secrets.SecretStore {
	provider := func(logger *slog.Logger) (secrets.SecretProvider, error) {
		return &sm.SecretProviderMock{
			GetFunc: func(key string) (string, error) {
				j, err := json.Marshal(result)
				if err != nil {
					return "", err
				}

				return string(j), nil
			},
		}, nil
	}

	return secrets.NewSecretStore(
		map[secrets.Provider]func(*slog.Logger) (secrets.SecretProvider, error){
			secrets.ProviderLocal: provider,
		},
	)
}
