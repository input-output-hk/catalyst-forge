package schema

#TagStrategy:     string
#enumTagStrategy: #TagStrategyGitCommit
#TagStrategyGitCommit: #TagStrategy & {
	"commit"
}

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

// Deployment contains the configuration for the deployment of the project.
#Deployment: {
	// Modules contains the deployment modules to deploy.
	modules: [...#DeploymentModule] @go(Modules,[]DeploymentModule)
}

// DeploymentModule contains the configuration for a single deployment module.
#DeploymentModule: {
	// Container contains the name of the container holding the deployment code.
	container: string @go(Container)

	// Environment contains the environment to deploy the module to.
	environment: string @go(Environment)

	// Values contains the values to pass to the deployment module.
	values: _ @go(Values,any)

	// Version contains the version of the deployment module.
	version: string @go(Version)
}

// Global contains the global configuration for the blueprint.
#Global: {
	// CI contains the configuration for the CI system.
	// +optional
	ci?: #GlobalCI @go(CI)
}

// CI contains the configuration for the CI system.
#GlobalCI: {
	// Local defines the filters to use when simulating a local CI run.
	local: [...string] @go(Local,[]string)

	// Providers contains the configuration for the providers being used by the CI system.
	// +optional
	providers?: #Providers @go(Providers)

	// Registries contains the container registries to push images to.
	// +optional
	registries?: [...string] @go(Registries,[]string)

	// Secrets contains global secrets that will be passed to all targets.
	// +optional
	secrets?: [...#Secret] @go(Secrets,[]Secret)

	// Tagging contains the tagging configuration for the CI system.
	// +optional
	tagging?: #Tagging @go(Tagging)
}

// GlobalDeployment contains the configuration for the global deployment of projects.
#GlobalDeployment: {
	// Repo contains the URL of the GitOps repository.
	repo: string @go(Repo)

	// Root contains the root deployment directory in the GitOps repository.
	root: string @go(Root)
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

	// Deployment contains the configuration for the deployment of the project.
	// +optional
	deployment?: #Deployment @go(Deployment)
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
	// Maps contains mappings for Earthly secret names to JSON keys in the secret.
	// Mutually exclusive with Name.
	// +optional
	maps?: {
		[string]: string
	} @go(Maps,map[string]string)

	// Name contains the name of the Earthly secret to use.
	// Mutually exclusive with Maps.
	// +optional
	name?: null | string @go(Name,*string)

	// Optional determines if the secret is optional.
	// +optional
	optional?: null | bool @go(Optional,*bool)

	// Path contains the path to the secret.
	path?: null | string @go(Path,*string)

	// Provider contains the provider to use for the secret.
	provider?: null | string @go(Provider,*string)
}
version: "1.0"
#Tagging: {
	// Aliases contains the aliases to use for git tags.
	// +optional
	aliases?: {
		[string]: string
	} @go(Aliases,map[string]string)

	// Strategy contains the tagging strategy to use for containers.
	strategy: #TagStrategy & {
		"commit"
	} @go(Strategy)
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
