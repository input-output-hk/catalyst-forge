package blueprint

import (
	g "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/global"
	p "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/project"
)

version: "1.0"

// Blueprint contains the schema for blueprint files.
#Blueprint: {
	// Global contains the global configuration for the repository.
	global?: g.#Global

	// Project contains the configuration for the project.
	project?: p.#Project

	// Version defines the version of the blueprint schema being used.
	version: string & =~"^\\d+\\.\\d+"
}

