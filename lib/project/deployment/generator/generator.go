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
	mg     deployment.ManifestGenerator
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
	manifests, err := d.mg.Generate(m)
	if err != nil {
		return nil, fmt.Errorf("failed to generate manifest for module: %w", err)
	}

	return manifests, nil
}

// NewGenerator creates a new deployment generator.
func NewGenerator(mg deployment.ManifestGenerator, logger *slog.Logger) Generator {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	return Generator{
		logger: logger,
		mg:     mg,
	}
}
