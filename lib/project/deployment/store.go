package deployment

import (
	"fmt"
	"log/slog"

	"github.com/input-output-hk/catalyst-forge/lib/project/deployment/providers/kcl"
)

// Provider represents a manifest generator provider.
type Provider string

const (
	// ProviderKCL represents the KCL manifest generator provider.
	ProviderKCL Provider = "kcl"
)

// ManifestGeneratorStore is a store of manifest generator providers.
type ManifestGeneratorStore struct {
	store map[Provider]func(*slog.Logger) ManifestGenerator
}

// NewDefaultManifestGeneratorStore returns a new ManifestGeneratorStore with the default providers.
func NewDefaultManifestGeneratorStore() ManifestGeneratorStore {
	return ManifestGeneratorStore{
		store: map[Provider]func(*slog.Logger) ManifestGenerator{
			ProviderKCL: func(logger *slog.Logger) ManifestGenerator {
				return kcl.NewKCLManifestGenerator(logger)
			},
		},
	}
}

// NewManifestGeneratorStore returns a new ManifestGeneratorStore with the given providers.
func NewManifestGeneratorStore(store map[Provider]func(*slog.Logger) ManifestGenerator) ManifestGeneratorStore {
	return ManifestGeneratorStore{store: store}
}

// NewGenerator returns a new ManifestGenerator client for the given provider.
func (s ManifestGeneratorStore) NewGenerator(logger *slog.Logger, p Provider) (ManifestGenerator, error) {
	if f, ok := s.store[p]; ok {
		return f(logger), nil
	}

	return nil, fmt.Errorf("unknown deployment module type: %s", p)
}
