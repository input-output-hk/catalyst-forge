package common

#Secret: {
	// Maps contains mappings for Earthly secret names to JSON keys in the secret.
	// Mutually exclusive with Name.
	maps?: [string]: string

	// Name contains the name of the Earthly secret to use.
	// Mutually exclusive with Maps.
	name?: string

	// Optional determines if the secret is optional.
	optional?: bool

	// Path contains the path to the secret.
	path: string

	// Provider contains the provider to use for the secret.
	provider: string
}
