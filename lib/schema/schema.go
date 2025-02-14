package schema

import (
	"embed"
	"fmt"
	"io/fs"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/load"
	"github.com/Masterminds/semver/v3"
	"github.com/input-output-hk/catalyst-forge/lib/tools/version"
)

//go:embed cue.mod/module.cue
//go:embed common/*.cue
//go:embed global/*.cue
//go:embed global/providers/*.cue
//go:embed project/*.cue
//go:embed main.cue
var Module embed.FS

// SCHEMA_PACKAGE is the name of the package that contains the schema.
const SCHEMA_PACKAGE = "schema"

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

	version, err := version.GetVersion(v)
	if err != nil {
		return RawSchema{}, fmt.Errorf("failed to get schema version: %w", err)
	}

	return RawSchema{
		Value:   bv,
		Version: version,
	}, nil
}

// buildPackage builds a CUE package from the given source files.
func buildPackage(src map[string]load.Source, ctx *cue.Context) (cue.Value, error) {
	insts := load.Instances(nil, &load.Config{
		Dir:        "/",
		ModuleRoot: "/",
		Overlay:    src,
		Package:    SCHEMA_PACKAGE,
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
	err := fs.WalkDir(Module, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		f, err := Module.ReadFile(path)
		if err != nil {
			return err
		}

		files["/"+path] = load.FromBytes(f)

		return nil
	})

	return files, err
}
