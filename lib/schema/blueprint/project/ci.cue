package project

import "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/common"

#CI: {
	// Targets configures the individual targets that are run by the CI system.
	targets: [string]: #Target
}

// Target contains the configuration for a single target.
#Target: {
	// Args contains the arguments to pass to the target.
	args?: [string]: string

	// Platforms contains the platforms to run the target against.
	platforms?: [...string]

	// Privileged determines if the target should run in privileged mode.
	privileged?: bool

	// Retries contains the configuration for the retries of an Earthly target.
	retries?: common.#CIRetries

	// Secrets contains the secrets to pass to the target.
	secrets?: [...common.#Secret]
}
