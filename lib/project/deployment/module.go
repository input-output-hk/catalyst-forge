package deployment

import (
	"fmt"
	"log/slog"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/format"
	"github.com/input-output-hk/catalyst-forge/lib/project/project"
	"github.com/input-output-hk/catalyst-forge/lib/project/secrets"
	sp "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/project"
	"github.com/input-output-hk/catalyst-forge/lib/tools/git/repo"
)

// ModuleBundle represents a deployment module bundle.
type ModuleBundle struct {
	Bundle sp.ModuleBundle
	Raw    cue.Value
}

// Dump dumps the deployment module bundle to CUE source.
func (d *ModuleBundle) Dump() ([]byte, error) {
	src, err := format.Node(d.Raw.Syntax())
	if err != nil {
		return nil, fmt.Errorf("failed to format bundle: %w", err)
	}

	return src, nil
}

// FetchBundle fetches a deployment bundle from the given project and repository.
func FetchBundle(r repo.GitRepo, projectPath string, store secrets.SecretStore, logger *slog.Logger) (ModuleBundle, error) {
	exists, err := r.Exists(projectPath)
	if err != nil {
		return ModuleBundle{}, fmt.Errorf("could not check if project path exists: %w", err)
	} else if !exists {
		return ModuleBundle{}, fmt.Errorf("project path does not exist: %s", projectPath)
	}

	loader := project.NewDefaultProjectLoader(cuecontext.New(), store, logger, project.WithFs(r.WorkFs()))
	p, err := loader.Load("/" + projectPath)
	if err != nil {
		return ModuleBundle{}, fmt.Errorf("could not load project: %w", err)
	}

	if p.Blueprint.Project == nil || p.Blueprint.Project.Deployment == nil {
		return ModuleBundle{}, fmt.Errorf("project does not have a deployment bundle")
	}

	return NewModuleBundle(&p), nil
}

// NewModuleBundle creates a new deployment module bundle from a project.
func NewModuleBundle(p *project.Project) ModuleBundle {
	bundle := p.Blueprint.Project.Deployment.Bundle
	raw := p.RawBlueprint.Get("project.deployment.bundle")
	return ModuleBundle{
		Bundle: bundle,
		Raw:    raw,
	}
}

// DumpModule dumps the deployment module to CUE source.
func DumpModule(ctx *cue.Context, mod sp.Module) ([]byte, error) {
	v := ctx.Encode(mod)

	if v.Err() != nil {
		return nil, fmt.Errorf("failed to encode module: %w", v.Err())
	}

	src, err := format.Node(v.Syntax())
	if err != nil {
		return nil, fmt.Errorf("failed to format module: %w", err)
	}

	return src, nil
}

// DumpBundle dumps the deployment module bundle to CUE source.
func DumpBundle(ctx *cue.Context, mod sp.ModuleBundle) ([]byte, error) {
	v := ctx.Encode(mod)

	if v.Err() != nil {
		return nil, fmt.Errorf("failed to encode bundle: %w", v.Err())
	}

	src, err := format.Node(v.Syntax())
	if err != nil {
		return nil, fmt.Errorf("failed to format bundle: %w", err)
	}

	return src, nil
}

// ParseModule parses a deployment module from CUE source.
func ParseBundle(ctx *cue.Context, src []byte) (ModuleBundle, error) {
	v := ctx.CompileBytes(src)
	if v.Err() != nil {
		return ModuleBundle{}, fmt.Errorf("failed to compile bundle: %w", v.Err())
	}

	return ParseBundleValue(v)
}

// ParseBundleValue parses a deployment module from a CUE value.
func ParseBundleValue(v cue.Value) (ModuleBundle, error) {
	var bundle sp.ModuleBundle
	if err := v.Decode(&bundle); err != nil {
		return ModuleBundle{}, fmt.Errorf("failed to decode bundle: %w", err)
	}

	return ModuleBundle{
		Bundle: bundle,
		Raw:    v,
	}, nil
}

// Validate validates a deployment module.
func Validate(mod sp.Module) error {
	if mod.Path == "" {
		if mod.Name == "" || mod.Registry == "" || mod.Version == "" {
			return fmt.Errorf("module must have at least one of (name, registry, version) or path")
		}
	}

	return nil
}
