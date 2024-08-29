package secrets

import (
	"fmt"
	"log/slog"

	"github.com/input-output-hk/catalyst-forge/forge/cli/pkg/secrets/providers"
)

// SecretStore is a store of secret providers.
type SecretStore struct {
	store map[Provider]func(*slog.Logger) (SecretProvider, error)
}

// NewDefaultSecretStore returns a new SecretStore with the default providers.
func NewDefaultSecretStore() SecretStore {
	return SecretStore{
		store: map[Provider]func(*slog.Logger) (SecretProvider, error){
			ProviderLocal: func(logger *slog.Logger) (SecretProvider, error) {
				return providers.NewLocalClient(logger)
			},
			ProviderAWS: func(logger *slog.Logger) (SecretProvider, error) {
				return providers.NewDefaultAWSClient(logger)
			},
		},
	}
}

// NewSecretStore returns a new SecretStore with the given providers.
func NewSecretStore(store map[Provider]func(*slog.Logger) (SecretProvider, error)) SecretStore {
	return SecretStore{store: store}
}

// NewClient returns a new SecretProvider client for the given provider.
func (s SecretStore) NewClient(logger *slog.Logger, p Provider) (SecretProvider, error) {
	if f, ok := s.store[p]; ok {
		return f(logger)
	}

	return nil, fmt.Errorf("unknown secret provider: %s", p)
}
