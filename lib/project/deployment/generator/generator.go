package generator

import (
	"fmt"
	"io"
	"log/slog"

	"cuelang.org/go/cue"
	"github.com/input-output-hk/catalyst-forge/lib/project/deployment"
	sp "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/project"
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
func (d *Generator) GenerateBundle(b deployment.ModuleBundle, env cue.Value) (GeneratorResult, error) {
	v := b.Raw.Unify(env)
	if v.Err() != nil {
		return GeneratorResult{}, fmt.Errorf("failed to unify bundle with environment: %w", v.Err())
	}

	nb, err := deployment.ParseBundleValue(v)
	if err != nil {
		return GeneratorResult{}, fmt.Errorf("failed to decode unified bundle value: %w", err)
	}

	bundle, err := b.Dump()
	if err != nil {
		return GeneratorResult{}, fmt.Errorf("failed to dump bundle: %w", err)
	}

	results := make(map[string][]byte)
	for name, module := range nb.Bundle.Modules {
		d.logger.Debug("Generating module", "name", name)
		raw := nb.Raw.LookupPath(cue.ParsePath(fmt.Sprintf("modules.%s", name)))
		result, err := d.Generate(module, raw, nb.Bundle.Env)
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
func (d *Generator) Generate(m sp.Module, raw cue.Value, env string) ([]byte, error) {
	if err := deployment.Validate(m); err != nil {
		return nil, fmt.Errorf("failed to validate module: %w", err)
	}

	mg, err := d.store.NewGenerator(d.logger, deployment.Provider(m.Type))
	if err != nil {
		return nil, fmt.Errorf("failed to get generator for module: %w", err)
	}

	manifests, err := mg.Generate(m, raw, env)
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
