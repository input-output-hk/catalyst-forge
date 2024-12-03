// Code generated by cue get go. DO NOT EDIT.

//cue:generate cue get go github.com/input-output-hk/catalyst-forge/lib/project/schema

package schema

// Providers contains the configuration for the providers being used by the CI system.
#Providers: {
	// AWS contains the configuration for the AWS provider.
	// +optional
	aws?: #ProviderAWS @go(AWS)

	// CUE contains the configuration for the CUE provider.
	// +optional
	cue?: #ProviderCue @go(CUE)

	// Docker contains the configuration for the DockerHub provider.
	// +optional
	docker?: #ProviderDocker @go(Docker)

	// Earthly contains the configuration for the Earthly Cloud provider.
	// +optional
	earthly?: #ProviderEarthly @go(Earthly)

	// Git contains the configuration for the Git provider.
	// +optional
	git?: #ProviderGit @go(Git)

	// Github contains the configuration for the Github provider.
	// +optional
	github?: #ProviderGithub @go(Github)

	// Timoni contains the configuration for the Timoni provider.
	// +optional
	timoni?: #TimoniProvider @go(Timoni)
}

// ProviderAWS contains the configuration for the AWS provider.
#ProviderAWS: {
	// Role contains the role to assume.
	role?: null | string @go(Role,*string)

	// Region contains the region to use.
	region?: null | string @go(Region,*string)

	// Registry contains the ECR registry to use.
	// +optional
	registry?: null | string @go(Registry,*string)
}

// ProviderCue contains the configuration for the CUE provider.
#ProviderCue: {
	// Install contains whether to install CUE in the CI environment.
	// +optional
	install?: null | bool @go(Install,*bool)

	// Registries contains the registries to use for publishing CUE modules
	registries: [...string] @go(Registries,[]string)

	// The version of CUE to use in CI.
	// +optional
	version?: string @go(Version)
}

// ProviderDocker contains the configuration for the DockerHub provider.
#ProviderDocker: {
	// Credentials contains the credentials to use for DockerHub
	credentials: #Secret @go(Credentials)
}

// ProviderEarthly contains the configuration for the Earthly Cloud provider.
#ProviderEarthly: {
	// Credentials contains the credentials to use for Earthly Cloud
	// +optional
	credentials?: #Secret @go(Credentials)

	// Org specifies the Earthly Cloud organization to use.
	// +optional
	org?: null | string @go(Org,*string)

	// Satellite contains the satellite to use for caching.
	// +optional
	satellite?: null | string @go(Satellite,*string)

	// The version of Earthly to use in CI.
	// +optional
	version?: null | string @go(Version,*string)
}

// ProviderGit contains the configuration for the Git provider.
#ProviderGit: {
	// Credentials contains the credentials to use for interacting with private repositories.
	// +optional
	credentials?: null | #Secret @go(Credentials,*Secret)
}

// ProviderGithub contains the configuration for the Github provider.
#ProviderGithub: {
	// Credentials contains the credentials to use for Github
	//  +optional
	credentials?: #Secret @go(Credentials)

	// Registry contains the Github registry to use.
	// +optional
	registry?: null | string @go(Registry,*string)
}

// TimoniProvider contains the configuration for the Timoni provider.
#TimoniProvider: {
	// Install contains whether to install Timoni in the CI environment.
	// +optional
	install?: null | bool @go(Install,*bool)

	// Registries contains the registries to use for publishing Timoni modules
	registries: [...string] @go(Registries,[]string)

	// The version of Timoni to use in CI.
	// +optional
	version?: string @go(Version)
}
