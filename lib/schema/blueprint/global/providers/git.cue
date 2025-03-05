package providers

import "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/common"

#Git: {
    // Credentials contains the credentials to use for interacting with private repositories.
    credentials: common.#Secret
}