package deployment

import (
	"fmt"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/format"
	"github.com/input-output-hk/catalyst-forge/lib/project/project"
	"github.com/input-output-hk/catalyst-forge/lib/project/schema"
)

// DeploymentBundle represents a deployment module bundle.
type DeploymentBundle struct {
	Bundle schema.DeploymentModuleBundle
	Raw    cue.Value
}

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
func DumpModule(mod schema.DeploymentModule) ([]byte, error) {
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
func DumpBundle(mod schema.DeploymentModuleBundle) ([]byte, error) {
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
func ParseBundle(src []byte) (schema.DeploymentModuleBundle, error) {
	ctx := cuecontext.New()
	v := ctx.CompileBytes(src)
	if v.Err() != nil {
		return schema.DeploymentModuleBundle{}, fmt.Errorf("failed to compile bundle: %w", v.Err())
	}

	var bundle schema.DeploymentModuleBundle
	if err := v.Decode(&bundle); err != nil {
		return schema.DeploymentModuleBundle{}, fmt.Errorf("failed to decode bundle: %w", err)
	}

	return bundle, nil
}

// Validate validates a deployment module.
func Validate(mod schema.DeploymentModule) error {
	if mod.Path == nil {
		if mod.Name == nil || mod.Registry == nil || mod.Version == nil {
			return fmt.Errorf("module must have at least one of (name, registry, version) or path")
		}
	}

	return nil
}
