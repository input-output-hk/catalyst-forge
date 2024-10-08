package schema

import (
	_ "embed"
)

type TagStrategy string

const (
	TagStrategyGitCommit TagStrategy = "commit"
)

//go:generate go run cuelang.org/go/cmd/cue@v0.9.2 get go --package schema --local .
//go:generate go run cuelang.org/go/cmd/cue@v0.9.2 def -fo _embed/schema.cue

//go:embed _embed/schema.cue
var RawSchemaFile []byte

// Blueprint contains the schema for blueprint files.
type Blueprint struct {
	// Version defines the version of the blueprint schema being used.
	Version string `json:"version"`

	// Global contains the global configuration for the blueprint.
	// +optional
	Global Global `json:"global"`

	// Project contains the configuration for the project.
	// +optional
	Project Project `json:"project"`
}

// Secret contains the secret provider and a list of mappings
type Secret struct {
	// Maps contains mappings for Earthly secret names to JSON keys in the secret.
	// Mutually exclusive with Name.
	// +optional
	Maps map[string]string `json:"maps"`

	// Name contains the name of the Earthly secret to use.
	// Mutually exclusive with Maps.
	// +optional
	Name *string `json:"name"`

	// Optional determines if the secret is optional.
	// +optional
	Optional *bool `json:"optional"`

	// Path contains the path to the secret.
	Path string `json:"path"`

	// Provider contains the provider to use for the secret.
	Provider string `json:"provider"`
}
