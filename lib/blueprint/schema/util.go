package schema

import (
	"fmt"

	"cuelang.org/go/cue"
	"github.com/Masterminds/semver/v3"
	"github.com/input-output-hk/catalyst-forge/lib/blueprint/pkg/version"
	cuetools "github.com/input-output-hk/catalyst-forge/lib/tools/pkg/cue"
)

// SchemaFile contains the schema for blueprint files.
type SchemaFile struct {
	Value   cue.Value
	Version *semver.Version
}

// Unify unifies the schema with the given value.
func (s SchemaFile) Unify(v cue.Value) cue.Value {
	return s.Value.Unify(v)
}

// LoadSchema loads the schema from the embedded schema file.
func LoadSchema(ctx *cue.Context) (SchemaFile, error) {
	v, err := cuetools.Compile(ctx, RawSchemaFile)
	if err != nil {
		return SchemaFile{}, err
	}

	version, err := version.GetVersion(v)
	if err != nil {
		return SchemaFile{}, fmt.Errorf("failed to get schema version: %w", err)
	}

	return SchemaFile{
		Value:   v.LookupPath(cue.ParsePath("#Blueprint")),
		Version: version,
	}, nil
}
