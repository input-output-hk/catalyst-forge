package providers

import "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/common"

#Github: {
    // Credentials contains the credentials to use for Github
    credentials?: common.#Secret

    // Registry contains the Github registry to use.
    registry?: string
}