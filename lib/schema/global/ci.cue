package global

import (
	"github.com/input-output-hk/catalyst-forge/lib/schema/common"
	p "github.com/input-output-hk/catalyst-forge/lib/schema/global/providers"
)

// CI contains the configuration for the CI system.
#CI: {
	// Local defines the filters to use when simulating a local CI run.
	local: [...string]

	// Providers contains the configuration for the providers being used by the CI system.
	providers?: p.#Providers

	// Registries contains the container registries to push images to.
	registries?: [...string]

	// Secrets contains global secrets that will be passed to all targets.
	secrets?:[...common.#Secret]
}

