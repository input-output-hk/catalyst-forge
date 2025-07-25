// Code generated by "cue exp gengotypes"; DO NOT EDIT.

package providers

import (
	"github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/common"
)

type AWS struct {
	// ECR contains the configuration for AWS ECR.
	Ecr AWSECR `json:"ecr"`

	// Role contains the role to assume.
	Role string `json:"role"`

	// Region contains the region to use.
	Region string `json:"region"`
}

type AWSECR struct {
	// AutoCreate contains whether to automatically create ECR repositories.
	AutoCreate bool `json:"autoCreate,omitempty"`

	// Registry is the ECR registry to login to during CI operations.
	Registry string `json:"registry,omitempty"`
}

type CUE struct {
	// Install contains whether to install CUE in the CI environment.
	Install bool `json:"install,omitempty"`

	// Registry contains the CUE registry to use for publishing CUE modules.
	Registry string `json:"registry,omitempty"`

	// RegistryPrefix contains the prefix to use for CUE registries.
	RegistryPrefix string `json:"registryPrefix,omitempty"`

	// The version of CUE to use in CI.
	Version string `json:"version,omitempty"`
}

type Docker struct {
	// Credentials contains the credentials to use for DockerHub
	Credentials common.Secret `json:"credentials"`
}

type Earthly struct {
	// Satellite contains the configuration for a remote Earthly Satellite.
	Satellite *EarthlySatellite `json:"satellite,omitempty"`

	// Version contains the version of Earthly to install in CI.
	Version string `json:"version,omitempty"`
}

// EarthlySatellite contains the configuration for a remote Earthly Satellite.
type EarthlySatellite struct {
	// Credentials contains the credentials to use for connecting to a remote
	// Earthly Satellite.
	Credentials *common.Secret `json:"credentials,omitempty"`
}

type Git struct {
	// Credentials contains the credentials to use for interacting with private repositories.
	Credentials common.Secret `json:"credentials"`
}

type Github struct {
	// Credentials contains the credentials to use for Github
	Credentials *common.Secret `json:"credentials,omitempty"`

	// Registry contains the Github registry to use.
	Registry string `json:"registry,omitempty"`
}

type KCL struct {
	// Install contains whether to install KCL in the CI environment.
	Install bool `json:"install,omitempty"`

	// Registries contains the registries to use for publishing KCL modules
	Registries []string `json:"registries"`

	// Version contains the version of KCL to install in CI.
	Version string `json:"version,omitempty"`
}

type Providers struct {
	// AWS contains the configuration for the AWS provider.
	Aws *AWS `json:"aws,omitempty"`

	// CUE contains the configuration for the CUE provider.
	Cue *CUE `json:"cue,omitempty"`

	// Docker contains the configuration for the DockerHub provider.
	Docker *Docker `json:"docker,omitempty"`

	// Earthly contains the configuration for the Earthly Cloud provider.
	Earthly *Earthly `json:"earthly,omitempty"`

	// Git contains the configuration for the Git provider.
	Git *Git `json:"git,omitempty"`

	// Github contains the configuration for the Github provider.
	Github *Github `json:"github,omitempty"`

	// KCL contains the configuration for the KCL provider.
	Kcl *KCL `json:"kcl,omitempty"`

	// Tailscale contains the configuration for the Tailscale provider.
	Tailscale *Tailscale `json:"tailscale,omitempty"`

	// Timoni contains the configuration for the Timoni provider.
	Timoni *Timoni `json:"timoni,omitempty"`
}

type Tailscale struct {
	// Credentials contains the OAuth2 credentials for authenticating to the
	// Tailscale network.
	Credentials *common.Secret `json:"credentials,omitempty"`

	// Tags is a comma-separated list of tags to impersonate.
	Tags string `json:"tags,omitempty"`

	// Version contains the version of Tailscale to install.
	Version string `json:"version,omitempty"`
}

type Timoni struct {
	// Install contains whether to install Timoni in the CI environment.
	Install bool `json:"install,omitempty"`

	// Registries contains the registries to use for publishing Timoni modules
	Registries []string `json:"registries"`

	// Version contains the version of Timoni to install in CI.
	Version string `json:"version,omitempty"`
}
