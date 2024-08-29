package schema

import (
	_ "embed"
)

//go:generate go run cuelang.org/go/cmd/cue@v0.9.2 get go --package schema --local .
//go:generate go run cuelang.org/go/cmd/cue@v0.9.2 def -fo _embed/schema.cue

//go:embed _embed/schema.cue
var RawSchemaFile []byte

// Blueprint contains the schema for blueprint files.
type Blueprint struct {
	Version  string            `json:"version"`
	Global   Global            `json:"global"`
	Registry string            `json:"registry"`
	Secrets  map[string]Secret `json:"secrets"`
	Targets  map[string]Target `json:"targets"`
}

// Global contains the global configuration.
type Global struct {
	Satellite string `json:"satellite"`
}

// Secret contains the secret provider and a list of mappings
type Secret struct {
	Path     string            `json:"path"`
	Provider string            `json:"provider"`
	Maps     map[string]string `json:"maps"`
}

// Target contains the configuration for a single target.
type Target struct {
	Args       map[string]string `json:"args"`
	Privileged bool              `json:"privileged"`
	Retries    int               `json:"retries"`
	Secrets    []Secret          `json:"secrets"`
}
