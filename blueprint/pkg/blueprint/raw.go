package blueprint

import (
	"cuelang.org/go/cue"
	"github.com/input-output-hk/catalyst-forge/blueprint/schema"
)

// RawBlueprint represents a raw (undecoded) blueprint.
type RawBlueprint struct {
	value cue.Value
}

// Decode decodes the raw blueprint into a schema.Blueprint.
func (r *RawBlueprint) Decode() (schema.Blueprint, error) {
	var cfg schema.Blueprint
	if err := r.value.Decode(&cfg); err != nil {
		return schema.Blueprint{}, err
	}

	return cfg, nil
}

// DecodePath decodes a path from the raw blueprint to the given interface.
func (r RawBlueprint) DecodePath(path string, x interface{}) error {
	v := r.Get(path)
	return v.Decode(x)
}

// Get returns a value from the raw blueprint.
func (r RawBlueprint) Get(path string) cue.Value {
	return r.value.LookupPath(cue.ParsePath(path))
}

// MarshalJSON marshals the raw blueprint into JSON.
func (r RawBlueprint) MarshalJSON() ([]byte, error) {
	return r.value.MarshalJSON()
}

// Value returns the raw blueprint value.
func (r RawBlueprint) Value() cue.Value {
	return r.value
}

// NewRawBlueprint creates a new raw blueprint.
func NewRawBlueprint(v cue.Value) RawBlueprint {
	return RawBlueprint{value: v}
}
