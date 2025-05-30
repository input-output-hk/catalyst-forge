// Code generated by "cue exp gengotypes"; DO NOT EDIT.

package blueprint

import (
	"github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/global"
	"github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/project"
)

// Blueprint contains the schema for blueprint files.
type Blueprint struct {
	// Data contains arbitrary configuration data that can be re-used throughout the blueprint.
	// This is useful for loading attribute values, for example:
	//
	//	data: foo: _ @env(name="FOO", type="string")
	//
	// This is a necessary workaround as attribute values cannot be arbitrarily referenced in CUE.
	// This field is otherwise unused.
	Data any/* CUE top */ `json:"data,omitempty"`

	// Global contains the global configuration for the repository.
	Global *global.Global `json:"global,omitempty"`

	// Project contains the configuration for the project.
	Project *project.Project `json:"project,omitempty"`

	// DEPRECATED: This field is deprecated and will be removed in a future version.
	Version string `json:"version,omitempty"`
}
