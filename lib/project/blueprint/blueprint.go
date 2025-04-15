package blueprint

import (
	"fmt"

	"cuelang.org/go/cue"
	cuetools "github.com/input-output-hk/catalyst-forge/lib/tools/cue"
)

// BlueprintFile represents a single blueprint file.
type BlueprintFile struct {
	Path  string
	Value cue.Value
}

// BlueprintFiles represents a collection of blueprint files.
type BlueprintFiles []BlueprintFile

// Unify unifies the blueprints into a single CUE value. If the unification
// fails, an error is returned.
func (b BlueprintFiles) Unify(ctx *cue.Context) (cue.Value, error) {
	v := ctx.CompileString("{}")
	for _, bp := range b {
		v = v.Unify(bp.Value)
	}

	if err := cuetools.Validate(v); err != nil {
		return cue.Value{}, err
	}

	return v, nil
}

// NewBlueprintFile creates a new BlueprintFile from the given CUE context,
// path, and contents. The contents are compiled and validated, including
// injecting any necessary environment variables. Additionally, the version is
// extracted from the CUE value. If the version is not found or invalid, or the
// final CUE value is invalid, an error is returned.
func NewBlueprintFile(ctx *cue.Context, path string, contents []byte) (BlueprintFile, error) {
	v, err := cuetools.Compile(ctx, contents)
	if err != nil {
		return BlueprintFile{}, fmt.Errorf("failed to compile CUE file: %w", err)
	}

	return BlueprintFile{
		Path:  path,
		Value: v,
	}, nil
}
