package blueprint

import "cuelang.org/go/cue"

// RawBlueprint represents a raw (undecoded) blueprint.
type RawBlueprint struct {
	value cue.Value
}

// Decode decodes the raw blueprint into the given value.
func (r RawBlueprint) Decode(x interface{}) error {
	return r.value.Decode(x)
}

// DecodePath decodes a value from the raw blueprint.
func (r RawBlueprint) DecodePath(path string, x interface{}) error {
	v := r.Get(path)
	return v.Decode(x)
}

// Get returns a value from the raw blueprint.
func (r RawBlueprint) Get(path string) cue.Value {
	return r.value.LookupPath(cue.ParsePath(path))
}

// Value returns the raw blueprint value.
func (r RawBlueprint) Value() cue.Value {
	return r.value
}

// NewRawBlueprint creates a new raw blueprint.
func NewRawBlueprint(v cue.Value) RawBlueprint {
	return RawBlueprint{value: v}
}
