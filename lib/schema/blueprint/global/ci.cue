package global

import (
	"github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/common"
	p "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/global/providers"
)

// CI contains the configuration for the CI system.
#CI: {
	// Local defines the filters to use when simulating a local CI run.
	local: [...string]

	// Providers contains the configuration for the providers being used by the CI system.
	providers?: p.#Providers

	// Registries contains the container registries to push images to.
	registries?: [...string]

	// Release contains the configuration for the release of a project.
	release?: #Release

	// Retries contains the configuration for the retries of an Earthly target.
	retries?: common.#CIRetries

	// Secrets contains global secrets that will be passed to all targets.
	secrets?: [...common.#Secret]
}
