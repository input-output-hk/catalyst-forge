package providers

import "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/common"

#Earthly: {
	// Satellite contains the configuration for a remote Earthly Satellite.
	satellite?: #EarthlySatellite

	// Version contains the version of Earthly to install in CI.
	version?: string
}

// EarthlySatellite contains the configuration for a remote Earthly Satellite.
#EarthlySatellite: {
	// Credentials contains the credentials to use for connecting to a remote
	// Earthly Satellite.
	credentials?: common.#Secret
}
