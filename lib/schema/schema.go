package schema

import (
	"fmt"
	"io/fs"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/load"
	"github.com/Masterminds/semver/v3"
	"github.com/input-output-hk/catalyst-forge/lib/schema/blueprint"
)

// RawSchema represents the raw blueprint schema loaded from the embedded module.
type RawSchema struct {
	Value   cue.Value
	Version *semver.Version
}

// Unify unifies the schema with the given value.
func (r RawSchema) Unify(v cue.Value) cue.Value {
	return r.Value.Unify(v)
}

// LoadSchema loads the blueprint schema from the embedded module.
func LoadSchema(ctx *cue.Context) (RawSchema, error) {
	files, err := loadSrcFiles()
	if err != nil {
		return RawSchema{}, err
	}

	v, err := buildPackage(files, ctx)
	if err != nil {
		return RawSchema{}, err
	}

	bv := v.LookupPath(cue.ParsePath("#Blueprint"))
	if bv.Err() != nil {
		return RawSchema{}, fmt.Errorf("failed to load schema: %w", bv.Err())
	}

	return RawSchema{
		Value: bv,
	}, nil
}

// buildPackage builds a CUE package from the given source files.
func buildPackage(src map[string]load.Source, ctx *cue.Context) (cue.Value, error) {
	insts := load.Instances(nil, &load.Config{
		Dir:        "/",
		ModuleRoot: "/",
		Overlay:    src,
		Package:    blueprint.SCHEMA_PACKAGE,
	})

	v := ctx.BuildInstance(insts[0])
	if v.Err() != nil {
		return cue.Value{}, v.Err()
	}

	return v, nil
}

// loadSrcFiles loads the source files from the embedded module.
func loadSrcFiles() (map[string]load.Source, error) {
	files := map[string]load.Source{}
	err := fs.WalkDir(blueprint.Module, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		f, err := blueprint.Module.ReadFile(path)
		if err != nil {
			return err
		}

		files["/"+path] = load.FromBytes(f)

		return nil
	})

	return files, err
}
