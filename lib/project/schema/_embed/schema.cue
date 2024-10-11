package schema

// Global contains the global configuration for the blueprint.
#Global: {
	// CI contains the configuration for the CI system.
	// +optional
	ci?: #GlobalCI @go(CI)

	// Deployment contains the global configuration for the deployment of projects.
	// +optional
	deployment?: #GlobalDeployment @go(Deployment)
}

// CI contains the configuration for the CI system.
#GlobalCI: {
	// DefaultBranch contains the default branch of the repository.
	// +optional
	defaultBranch?: null | string @go(DefaultBranch,*string)

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
	// Registry contains the URL of the container registry holding the deployment code.
	registry: string @go(Registry)

	// Repo contains the configuration for the global deployment repository.
	repo: #GlobalDeploymentRepo @go(Repo)

	// Root contains the root deployment directory in the deployment repository.
	root: string @go(Root)
}

// GlobalDeploymentRepo contains the configuration for the global deployment repository.
#GlobalDeploymentRepo: {
	// Ref contains the ref to use for the deployment repository.
	ref: string @go(Ref)

	// URL contains the URL of the deployment repository.
	url: string @go(Url)
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

	// Git contains the configuration for the Git provider.
	// +optional
	git?: #ProviderGit @go(Git)

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

// ProviderGit contains the configuration for the Git provider.
#ProviderGit: {
	// Credentials contains the credentials to use for interacting with private repositories.
	// +optional
	credentials?: null | #Secret @go(Credentials,*Secret)
}
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
	// Environment contains the environment to deploy the module to.
	environment: (_ | *"dev") & {
		string
	} @go(Environment)

	// Modules contains the configuration for the deployment modules for the project.
	// +optional
	modules?: null | #DeploymentModules @go(Modules,*DeploymentModules)
}

// Deployment contains the configuration for the deployment of the project.
#DeploymentModules: {
	// Main contains the configuration for the main deployment module.
	main: #Module @go(Main)

	// Support contains the configuration for the support deployment modules.
	// +optional
	support?: {
		[string]: #Module
	} @go(Support,map[string]Module)
}
version: "1.0"

// Module contains the configuration for a deployment module.
#Module: {
	// Container contains the name of the container holding the deployment code.
	// Defaults to <module_name>-deployment). For the main module, <module_name> is the project name.
	// +optional
	container?: null | string @go(Container,*string)

	// Namespace contains the namespace to deploy the module to.
	namespace: (_ | *"default") & {
		string
	} @go(Namespace)

	// Values contains the values to pass to the deployment module.
	values: _ @go(Values,any)

	// Version contains the version of the deployment module.
	version: string @go(Version)
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

	// Release contains the configuration for the release of the project.
	// +optional
	release?: {
		[string]: #Release
	} @go(Release,map[string]Release)
}
#ProjectCI: {
	// Targets configures the individual targets that are run by the CI system.
	// +optional
	targets?: {
		[string]: #Target
	} @go(Targets,map[string]Target)
}

// Release contains the configuration for a project release.
#Release: {
	// Config contains the configuration to pass to the release.
	config: _ @go(Config,any)

	// On contains the events that trigger the release.
	on: [...string] @go(On,[]string)

	// Target is the Earthly target to run for this release.
	target: string @go(Target)

	// Type is the type of releaser to use.
	type: string @go(Type)
}
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
	path: string @go(Path)

	// Provider contains the provider to use for the secret.
	provider: string @go(Provider)
}
