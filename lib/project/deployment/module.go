package deployment

import (
	"fmt"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/format"
	"github.com/input-output-hk/catalyst-forge/lib/project/project"
	sp "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/project"
)

// DeploymentBundle represents a deployment module bundle.
type DeploymentBundle struct {
	Bundle sp.ModuleBundle
	Raw    cue.Value
}

// Dump dumps the deployment module bundle to CUE source.
func (d *DeploymentBundle) Dump() ([]byte, error) {
	src, err := format.Node(d.Raw.Syntax())
	if err != nil {
		return nil, fmt.Errorf("failed to format bundle: %w", err)
	}

	return src, nil
}

func NewDeploymentBundle(p project.Project) DeploymentBundle {
	bundle := p.Blueprint.Project.Deployment.Modules
	raw := p.RawBlueprint.Get("project.deployment.modules")
	return DeploymentBundle{
		Bundle: bundle,
		Raw:    raw,
	}
}

// DumpModule dumps the deployment module to CUE source.
func DumpModule(mod sp.Module) ([]byte, error) {
	ctx := cuecontext.New()
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
func DumpBundle(mod sp.ModuleBundle) ([]byte, error) {
	ctx := cuecontext.New()
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
func ParseBundle(src []byte) (sp.ModuleBundle, error) {
	ctx := cuecontext.New()
	v := ctx.CompileBytes(src)
	if v.Err() != nil {
		return sp.ModuleBundle{}, fmt.Errorf("failed to compile bundle: %w", v.Err())
	}

	var bundle sp.ModuleBundle
	if err := v.Decode(&bundle); err != nil {
		return sp.ModuleBundle{}, fmt.Errorf("failed to decode bundle: %w", err)
	}

	return bundle, nil
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
