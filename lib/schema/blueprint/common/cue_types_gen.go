// Code generated by "cue exp gengotypes"; DO NOT EDIT.

package common

type Secret struct {
	// Maps contains mappings for Earthly secret names to JSON keys in the secret.
	// Mutually exclusive with Name.
	Maps map[string]string `json:"maps,omitempty"`

	// Name contains the name of the Earthly secret to use.
	// Mutually exclusive with Maps.
	Name string `json:"name,omitempty"`

	// Optional determines if the secret is optional.
	Optional bool `json:"optional,omitempty"`

	// Path contains the path to the secret.
	Path string `json:"path"`

	// Provider contains the provider to use for the secret.
	Provider string `json:"provider"`
}
