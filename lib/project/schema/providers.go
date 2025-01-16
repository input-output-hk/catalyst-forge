package schema

// Providers contains the configuration for the providers being used by the CI system.
type Providers struct {
	// AWS contains the configuration for the AWS provider.
	// +optional
	AWS ProviderAWS `json:"aws"`

	// CUE contains the configuration for the CUE provider.
	// +optional
	CUE ProviderCue `json:"cue"`

	// Docker contains the configuration for the DockerHub provider.
	// +optional
	Docker ProviderDocker `json:"docker"`

	// Earthly contains the configuration for the Earthly Cloud provider.
	// +optional
	Earthly ProviderEarthly `json:"earthly"`

	// Git contains the configuration for the Git provider.
	// +optional
	Git ProviderGit `json:"git"`

	// Github contains the configuration for the Github provider.
	// +optional
	Github ProviderGithub `json:"github"`

	// KCL contains the configuration for the KCL provider.
	// +optional
	KCL ProviderKCL `json:"kcl"`

	// Timoni contains the configuration for the Timoni provider.
	// +optional
	Timoni TimoniProvider `json:"timoni"`
}

// ProviderAWS contains the configuration for the AWS provider.
type ProviderAWS struct {
	// ECR contains the configuration for AWS ECR.
	// +optional
	ECR ProviderAWSECR `json:"ecr"`

	// Role contains the role to assume.
	Role string `json:"role"`

	// Region contains the region to use.
	Region string `json:"region"`
}

type ProviderAWSECR struct {
	// AutoCreate contains whether to automatically create ECR repositories.
	// +optional
	AutoCreate *bool `json:"autoCreate"`

	// Registry is the ECR registry to login to during CI operations.
	// +optional
	Registry *string `json:"registry"`
}

// ProviderCue contains the configuration for the CUE provider.
type ProviderCue struct {
	// Install contains whether to install CUE in the CI environment.
	// +optional
	Install *bool `json:"install"`

	// Registry contains the CUE registry to use.
	Registry *string `json:"registry"`

	// RegistryPrefix contains the prefix to use for CUE registries.
	// +optional
	RegistryPrefix *string `json:"registryPrefix"`

	// The version of CUE to use in CI.
	// +optional
	Version string `json:"version"`
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

// ProviderGit contains the configuration for the Git provider.
type ProviderGit struct {
	// Credentials contains the credentials to use for interacting with private repositories.
	// +optional
	Credentials *Secret `json:"credentials"`
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

// ProviderKCL contains the configuration for the KCL provider.
type ProviderKCL struct {
	// Install contains whether to install KCL in the CI environment.
	// +optional
	Install *bool `json:"install"`

	// Registries contains the registries to use for publishing KCL modules
	Registries []string `json:"registries"`

	// The version of KCL to install in the CI environment
	// +optional
	Version string `json:"version"`
}

// TimoniProvider contains the configuration for the Timoni provider.
type TimoniProvider struct {
	// Install contains whether to install Timoni in the CI environment.
	// +optional
	Install *bool `json:"install"`

	// Registries contains the registries to use for publishing Timoni modules
	Registries []string `json:"registries"`

	// The version of Timoni to use in CI.
	// +optional
	Version string `json:"version"`
}
