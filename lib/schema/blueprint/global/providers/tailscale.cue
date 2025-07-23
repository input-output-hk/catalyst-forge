package providers

import "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/common"

#Tailscale: {
	// Credentials contains the OAuth2 credentials for authenticating to the
	// Tailscale network.
	credentials?: common.#Secret

	// Tags is a comma-separated list of tags to impersonate.
	tags?: string

	// Version contains the version of Tailscale to install.
	version?: string
}
