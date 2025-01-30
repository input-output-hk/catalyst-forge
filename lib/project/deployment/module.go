package deployment

import (
	"fmt"

	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/format"
	"github.com/input-output-hk/catalyst-forge/lib/project/schema"
)

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
