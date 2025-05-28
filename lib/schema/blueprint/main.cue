package blueprint

import (
	g "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/global"
	p "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/project"
)

// Blueprint contains the schema for blueprint files.
#Blueprint: {
	// Data contains arbitrary configuration data that can be re-used throughout the blueprint.
	// This is useful for loading attribute values, for example:
	//   data: foo: _ @env(name="FOO", type="string")
	// This is a necessary workaround as attribute values cannot be arbitrarily referenced in CUE.
	// This field is otherwise unused.
	data?: _

	// Global contains the global configuration for the repository.
	global?: g.#Global

	// Project contains the configuration for the project.
	project?: p.#Project

	// DEPRECATED: This field is deprecated and will be removed in a future version.
	version?: string
}
