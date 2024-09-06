package schema

// Blueprint contains the schema for blueprint files.
#Blueprint: {
	// Version defines the version of the blueprint schema being used.
	version: =~"^\\d+\\.\\d+" @go(Version)

	// Global contains the global configuration for the blueprint.
	// +optional
	global?: #Global @go(Global)

	// Project contains the configuration for the project.
	// +optional
	project?: #Project @go(Project)
}

// Global contains the global configuration for the blueprint.
#Global: {
	// CI contains the configuration for the CI system.
	// +optional
	ci?: #GlobalCI @go(CI)
}

// Project contains the configuration for the project.
#Project: {
	// Name contains the name of the project.
	name: =~"^[a-z][a-z0-9_-]*$" @go(Name)

	// Container is the name that the container will be built as.
	container: (_ | *name) & {
		string
	} @go(Container)

	// CI contains the configuration for the CI system.
	// +optional
	ci?: #ProjectCI @go(CI)
}

// CI contains the configuration for the CI system.
#GlobalCI: {
	// Providers contains the configuration for the providers being used by the CI system.
	// +optional
	providers?: #Providers @go(Providers)

	// Registries contains the container registries to push images to.
	// +optional
	registries?: [...string] @go(Registries,[]string)

	// Tagging contains the tagging configuration for the CI system.
	// +optional
	tagging?: #Tagging @go(Tagging)
}
#ProjectCI: {
	// Targets configures the individual targets that are run by the CI system.
	// +optional
	targets?: {
		[string]: #Target
	} @go(Targets,map[string]Target)
}

// Providers contains the configuration for the providers being used by the CI system.
#Providers: {
	// AWS contains the configuration for the AWS provider.
	// +optional
	aws?: #ProviderAWS @go(AWS)

	// Docker contains the configuration for the DockerHub provider.
	// +optional
	docker?: #ProviderDocker @go(Docker)

	// Earthly contains the configuration for the Earthly Cloud provider.
	// +optional
	earthly?: #ProviderEarthly @go(Earthly)

	// Github contains the configuration for the Github provider.
	// +optional
	github?: #ProviderGithub @go(Github)
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

// ProviderGithub contains the configuration for the Github provider.
#ProviderGithub: {
	// Credentials contains the credentials to use for Github
	//  +optional
	credentials?: #Secret @go(Credentials)

	// Registry contains the Github registry to use.
	// +optional
	registry?: null | string @go(Registry,*string)
}

// Secret contains the secret provider and a list of mappings
#Secret: {
	// Path contains the path to the secret.
	path?: null | string @go(Path,*string)

	// Provider contains the provider to use for the secret.
	provider?: null | string @go(Provider,*string)

	// Maps contains the mappings for the secret.
	// +optional
	maps?: {
		[string]: string
	} @go(Maps,map[string]string)
}
version: "1.0"
#Tagging: {
	// Aliases contains the aliases to use for git tags.
	// +optional
	aliases?: {
		[string]: string
	} @go(Aliases,map[string]string)

	// Strategy contains the tagging strategy to use for containers.
	strategy: "commit" @go(Strategy)
}

// Target contains the configuration for a single target.
#Target: {
	// Args contains the arguments to pass to the target.
	// +optional
	args?: {
		[string]: string
	} @go(Args,map[string]string)

	// Platforms contains the platforms to run the target against.
	// +optional
	platforms?: [...string] @go(Platforms,[]string)

	// Privileged determines if the target should run in privileged mode.
	// +optional
	privileged?: null | bool @go(Privileged,*bool)

	// Retries contains the number of times to retry the target.
	// +optional
	retries?: null | int @go(Retries,*int)

	// Secrets contains the secrets to pass to the target.
	// +optional
	secrets?: [...#Secret] @go(Secrets,[]Secret)
}
