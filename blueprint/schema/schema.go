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

// Global contains the global configuration for the blueprint.
type Global struct {
	// CI contains the configuration for the CI system.
	// +optional
	CI GlobalCI `json:"ci"`
}

// Project contains the configuration for the project.
type Project struct {
	// Name contains the name of the project.
	Name string `json:"name"`

	// Container is the name that the container will be built as.
	Container string `json:"container"`

	// CI contains the configuration for the CI system.
	// +optional
	CI ProjectCI `json:"ci"`
}

// CI contains the configuration for the CI system.
type GlobalCI struct {
	// Providers contains the configuration for the providers being used by the CI system.
	// +optional
	Providers Providers `json:"providers"`

	// Registries contains the container registries to push images to.
	// +optional
	Registries []string `json:"registries"`

	// Secrets contains global secrets that will be passed to all targets.
	// +optional
	Secrets []Secret `json:"secrets"`

	// Tagging contains the tagging configuration for the CI system.
	// +optional
	Tagging Tagging `json:"tagging"`
}

type ProjectCI struct {
	// Targets configures the individual targets that are run by the CI system.
	// +optional
	Targets map[string]Target `json:"targets"`
}

// Providers contains the configuration for the providers being used by the CI system.
type Providers struct {
	// AWS contains the configuration for the AWS provider.
	// +optional
	AWS ProviderAWS `json:"aws"`

	// Docker contains the configuration for the DockerHub provider.
	// +optional
	Docker ProviderDocker `json:"docker"`

	// Earthly contains the configuration for the Earthly Cloud provider.
	// +optional
	Earthly ProviderEarthly `json:"earthly"`

	// Github contains the configuration for the Github provider.
	// +optional
	Github ProviderGithub `json:"github"`
}

// ProviderAWS contains the configuration for the AWS provider.
type ProviderAWS struct {
	// Role contains the role to assume.
	Role *string `json:"role"`

	// Region contains the region to use.
	Region *string `json:"region"`

	// Registry contains the ECR registry to use.
	// +optional
	Registry *string `json:"registry"`
}

// ProviderDocker contains the configuration for the DockerHub provider.
type ProviderDocker struct {
	// Credentials contains the credentials to use for DockerHub
	Credentials Secret `json:"credentials"`
}

// ProviderEarthly contains the configuration for the Earthly Cloud provider.
type ProviderEarthly struct {
	// Credentials contains the credentials to use for Earthly Cloud
	// +optional
	Credentials Secret `json:"credentials"`

	// Org specifies the Earthly Cloud organization to use.
	// +optional
	Org *string `json:"org"`

	// Satellite contains the satellite to use for caching.
	// +optional
	Satellite *string `json:"satellite"`

	// The version of Earthly to use in CI.
	// +optional
	Version *string `json:"version"`
}

// ProviderGithub contains the configuration for the Github provider.
type ProviderGithub struct {
	// Credentials contains the credentials to use for Github
	//  +optional
	Credentials Secret `json:"credentials"`

	// Registry contains the Github registry to use.
	// +optional
	Registry *string `json:"registry"`
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
	Path *string `json:"path"`

	// Provider contains the provider to use for the secret.
	Provider *string `json:"provider"`
}

type Tagging struct {
	// Aliases contains the aliases to use for git tags.
	// +optional
	Aliases map[string]string `json:"aliases"`

	// Strategy contains the tagging strategy to use for containers.
	Strategy TagStrategy `json:"strategy"`
}

// Target contains the configuration for a single target.
type Target struct {
	// Args contains the arguments to pass to the target.
	// +optional
	Args map[string]string `json:"args"`

	// Platforms contains the platforms to run the target against.
	// +optional
	Platforms []string `json:"platforms"`

	// Privileged determines if the target should run in privileged mode.
	// +optional
	Privileged *bool `json:"privileged"`

	// Retries contains the number of times to retry the target.
	// +optional
	Retries *int `json:"retries"`

	// Secrets contains the secrets to pass to the target.
	// +optional
	Secrets []Secret `json:"secrets"`
}
