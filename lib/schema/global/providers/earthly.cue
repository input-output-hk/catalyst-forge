package providers

import "github.com/input-output-hk/catalyst-forge/lib/schema/common"

#Earthly: {
    // Credentials contains the credentials to use for Earthly Cloud
    credentials?: common.#Secret

    // Org specifies the Earthly Cloud organization to use.
    org?: string

    // Satellite contains the satellite to use for caching.
    satellite?: string

    // Version contains the version of Earthly to install in CI.
    version?: string
}