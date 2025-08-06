package deployment

import (
	"fmt"
	"log/slog"

	"github.com/input-output-hk/catalyst-forge/lib/deployment/providers/git"
	"github.com/input-output-hk/catalyst-forge/lib/deployment/providers/helm"
	"github.com/input-output-hk/catalyst-forge/lib/deployment/providers/kcl"
)

// Provider represents a manifest generator provider.
type Provider string

const (
	// ProviderGit represents the Git manifest generator provider.
	ProviderGit Provider = "git"

	// ProviderHelm represents the Helm manifest generator provider.
	ProviderHelm Provider = "helm"

	// ProviderKCL represents the KCL manifest generator provider.
	ProviderKCL Provider = "kcl"
)

// ManifestGeneratorStore is a store of manifest generator providers.
type ManifestGeneratorStore struct {
	store map[Provider]func(*slog.Logger) (ManifestGenerator, error)
}

// NewDefaultManifestGeneratorStore returns a new ManifestGeneratorStore with the default providers.
func NewDefaultManifestGeneratorStore() ManifestGeneratorStore {
	return ManifestGeneratorStore{
		store: map[Provider]func(*slog.Logger) (ManifestGenerator, error){
			ProviderGit: func(logger *slog.Logger) (ManifestGenerator, error) {
				return git.NewGitManifestGenerator(logger), nil
			},
			ProviderHelm: func(logger *slog.Logger) (ManifestGenerator, error) {
				return helm.NewHelmManifestGenerator(logger)
			},
			ProviderKCL: func(logger *slog.Logger) (ManifestGenerator, error) {
				return kcl.NewKCLManifestGenerator(logger)
			},
		},
	}
}

// NewManifestGeneratorStore returns a new ManifestGeneratorStore with the given providers.
func NewManifestGeneratorStore(store map[Provider]func(*slog.Logger) (ManifestGenerator, error)) ManifestGeneratorStore {
	return ManifestGeneratorStore{store: store}
}

// NewGenerator returns a new ManifestGenerator client for the given provider.
func (s ManifestGeneratorStore) NewGenerator(logger *slog.Logger, p Provider) (ManifestGenerator, error) {
	if f, ok := s.store[p]; ok {
		return f(logger)
	}

	return nil, fmt.Errorf("unknown deployment module type: %s", p)
}
