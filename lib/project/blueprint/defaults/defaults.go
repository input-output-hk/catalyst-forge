package defaults

import (
	"cuelang.org/go/cue"
)

// DefaultSetter is an interface for setting dynamic default values in a CUE value.
type DefaultSetter interface {
	SetDefault(v cue.Value) (cue.Value, error)
}

// GetDefaultSetters returns a list of all default setters.
func GetDefaultSetters() []DefaultSetter {
	return []DefaultSetter{
		DeploymentModuleSetter{},
		ReleaseTargetSetter{},
	}
}
