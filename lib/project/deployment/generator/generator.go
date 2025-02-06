package generator

import (
	"fmt"
	"io"
	"log/slog"

	"github.com/input-output-hk/catalyst-forge/lib/project/deployment"
	"github.com/input-output-hk/catalyst-forge/lib/project/schema"
)

// GeneratorResult is the result of a deployment generation.
type GeneratorResult struct {
	Manifests map[string][]byte
	Module    []byte
}

// Generator is a deployment generator.
type Generator struct {
	logger *slog.Logger
	store  deployment.ManifestGeneratorStore
}

// GenerateBundle generates manifests for a deployment bundle.
func (d *Generator) GenerateBundle(b schema.DeploymentModuleBundle) (GeneratorResult, error) {
	bundle, err := deployment.DumpBundle(b)
	if err != nil {
		return GeneratorResult{}, fmt.Errorf("failed to dump bundle: %w", err)
	}

	results := make(map[string][]byte)
	for name, module := range b {
		d.logger.Debug("Generating module", "name", name)
		result, err := d.Generate(module)
		if err != nil {
			return GeneratorResult{}, fmt.Errorf("failed to generate module %s: %w", name, err)
		}

		results[name] = result
	}

	return GeneratorResult{
		Manifests: results,
		Module:    bundle,
	}, nil
}

// Generate generates manifests for a deployment module.
func (d *Generator) Generate(m schema.DeploymentModule) ([]byte, error) {
	if err := deployment.Validate(m); err != nil {
		return nil, fmt.Errorf("failed to validate module: %w", err)
	}

	mg, err := d.store.NewGenerator(d.logger, deployment.Provider(m.Type))
	if err != nil {
		return nil, fmt.Errorf("failed to get generator for module: %w", err)
	}

	manifests, err := mg.Generate(m)
	if err != nil {
		return nil, fmt.Errorf("failed to generate manifest for module: %w", err)
	}

	return manifests, nil
}

// NewGenerator creates a new deployment generator.
func NewGenerator(store deployment.ManifestGeneratorStore, logger *slog.Logger) Generator {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	return Generator{
		logger: logger,
		store:  store,
	}
}
