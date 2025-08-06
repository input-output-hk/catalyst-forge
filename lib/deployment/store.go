package deployment

import (
	"fmt"
	"log/slog"

	"github.com/input-output-hk/catalyst-forge/lib/deployment/providers/git"
	"github.com/input-output-hk/catalyst-forge/lib/deployment/providers/helm"
	"github.com/input-output-hk/catalyst-forge/lib/deployment/providers/kcl"
	kclext "github.com/input-output-hk/catalyst-forge/lib/external/kcl"
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

// Option configures the ManifestGeneratorStore
type Option func(*ManifestGeneratorStore) error

// WithKCLOpts sets the KCL options for the store
func WithKCLOpts(opts ...kclext.Option) Option {
	return func(s *ManifestGeneratorStore) error {
		s.kclOpts = append(s.kclOpts, opts...)
		return nil
	}
}

// ManifestGeneratorStore is a store of manifest generator providers.
type ManifestGeneratorStore struct {
	store   map[Provider]func(*slog.Logger) (ManifestGenerator, error)
	kclOpts []kclext.Option
}

// NewDefaultManifestGeneratorStore returns a new ManifestGeneratorStore with the default providers.
func NewDefaultManifestGeneratorStore(opts ...Option) (ManifestGeneratorStore, error) {
	store := ManifestGeneratorStore{
		store: map[Provider]func(*slog.Logger) (ManifestGenerator, error){
			ProviderGit: func(logger *slog.Logger) (ManifestGenerator, error) {
				return git.NewGitManifestGenerator(logger), nil
			},
			ProviderHelm: func(logger *slog.Logger) (ManifestGenerator, error) {
				return helm.NewHelmManifestGenerator(logger)
			},
		},
	}

	for _, opt := range opts {
		if err := opt(&store); err != nil {
			return ManifestGeneratorStore{}, fmt.Errorf("failed to apply option: %w", err)
		}
	}

	kclOpts := store.kclOpts
	store.store[ProviderKCL] = func(logger *slog.Logger) (ManifestGenerator, error) {
		return kcl.NewKCLManifestGenerator(logger, kclOpts...)
	}

	return store, nil
}

// NewManifestGeneratorStore returns a new ManifestGeneratorStore with the given providers.
func NewManifestGeneratorStore(store map[Provider]func(*slog.Logger) (ManifestGenerator, error)) ManifestGeneratorStore {
	return ManifestGeneratorStore{
		store:   store,
		kclOpts: nil,
	}
}

// NewGenerator returns a new ManifestGenerator client for the given provider.
func (s ManifestGeneratorStore) NewGenerator(logger *slog.Logger, p Provider) (ManifestGenerator, error) {
	if f, ok := s.store[p]; ok {
		return f(logger)
	}

	return nil, fmt.Errorf("unknown deployment module type: %s", p)
}
