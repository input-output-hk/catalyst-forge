package schema

#DeploymentModuleBundle: {
	[string]: #DeploymentModule
}

// Global contains the global configuration for the blueprint.
#Global: {
	// CI contains the configuration for the CI system.
	// +optional
	ci?: #GlobalCI @go(CI)

	// Deployment contains the global configuration for the deployment of projects.
	// +optional
	deployment?: #GlobalDeployment @go(Deployment)

	// Repo contains the configuration for the GitHub repository.
	repo: #GlobalRepo @go(Repo)
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
}

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

	// KCL contains the configuration for the KCL provider.
	// +optional
	kcl?: #ProviderKCL @go(KCL)

	// Timoni contains the configuration for the Timoni provider.
	// +optional
	timoni?: #TimoniProvider @go(Timoni)
}

// ProviderAWS contains the configuration for the AWS provider.
#ProviderAWS: {
	// ECR contains the configuration for AWS ECR.
	// +optional
	ecr?: #ProviderAWSECR @go(ECR)

	// Role contains the role to assume.
	role: string @go(Role)

	// Region contains the region to use.
	region: string @go(Region)
}
#ProviderAWSECR: {
	// AutoCreate contains whether to automatically create ECR repositories.
	// +optional
	autoCreate?: null | bool @go(AutoCreate,*bool)

	// Registry is the ECR registry to login to during CI operations.
	// +optional
	registry?: null | string @go(Registry,*string)
}

// ProviderCue contains the configuration for the CUE provider.
#ProviderCue: {
	// Install contains whether to install CUE in the CI environment.
	// +optional
	install?: null | bool @go(Install,*bool)

	// Registry contains the CUE registry to use.
	registry?: null | string @go(Registry,*string)

	// RegistryPrefix contains the prefix to use for CUE registries.
	// +optional
	registryPrefix?: null | string @go(RegistryPrefix,*string)

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
	credentials?: null | #Secret @go(Credentials,*Secret)

	// Registry contains the Github registry to use.
	// +optional
	registry?: null | string @go(Registry,*string)
}

// ProviderKCL contains the configuration for the KCL provider.
#ProviderKCL: {
	// Install contains whether to install KCL in the CI environment.
	// +optional
	install?: null | bool @go(Install,*bool)

	// Registries contains the registries to use for publishing KCL modules
	registries: [...string] @go(Registries,[]string)

	// The version of KCL to install in the CI environment
	// +optional
	version?: string @go(Version)
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
	// On contains the events that trigger the deployment.
	on: {
		...
	} @go(On,map[string]any)
	environment: _ | *"dev"

	// Modules contains the deployment modules for the project.
	modules: #DeploymentModuleBundle @go(Modules)
}

// GlobalDeployment contains the configuration for the global deployment of projects.
#GlobalDeployment: {
	// Environment contains the default environment to deploy projects to.
	environment: (_ | *"dev") & {
		string
	} @go(Environment)

	// Registries contains the configuration for the global deployment registries.
	registries: #GlobalDeploymentRegistries @go(Registries)

	// Repo contains the configuration for the global deployment repository.
	repo: #GlobalDeploymentRepo @go(Repo)

	// Root contains the root deployment directory in the deployment repository.
	root: string @go(Root)
}

// GlobalDeploymentRegistries contains the configuration for the global deployment registries.
#GlobalDeploymentRegistries: {
	// Containers contains the default container registry to use for deploying containers.
	containers: string @go(Containers)

	// Modules contains the container registry that holds deployment modules.
	modules: string @go(Modules)
}

// GlobalDeploymentRepo contains the configuration for the global deployment repository.
#GlobalDeploymentRepo: {
	// Ref contains the ref to use for the deployment repository.
	ref: string @go(Ref)

	// URL contains the URL of the deployment repository.
	url: string @go(Url)
}
version: "1.0"

// Module contains the configuration for a deployment module.
#DeploymentModule: {
	// Instance contains the instance name to use for all generated resources.
	// +optional
	instance?: string @go(Instance)

	// Name contains the name of the module to deploy.
	// +optional
	name?: null | string @go(Name,*string)

	// Namespace contains the namespace to deploy the module to.
	namespace: (_ | *"default") & {
		string
	} @go(Namespace)

	// Path contains the path to the module.
	// +optional
	path?: null | string @go(Path,*string)

	// Registry contains the registry to pull the module from.
	// +optional
	registry?: null | string @go(Registry,*string)

	// Type contains the type of the module.
	type: (_ | *"kcl") & {
		string
	} @go(Type)

	// Values contains the values to pass to the deployment module.
	values: _ @go(Values,any)

	// Version contains the version of the deployment module.
	// +optional
	version?: null | string @go(Version,*string)
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
	// +optional
	config?: _ @go(Config,any)

	// On contains the events that trigger the release.
	on: {
		...
	} @go(On,map[string]any)

	// Target is the Earthly target to run for this release.
	// Defaults to release name.
	// +optional
	target?: string @go(Target)
}
#Tagging: {
	strategy: "commit"
}
#GlobalRepo: {
	// Name contains the name of the repository (e.g. "owner/repo-name").
	name: string @go(Name)

	// DefaultBranch contains the default branch of the repository.
	defaultBranch: string @go(DefaultBranch)
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

// TimoniProvider contains the configuration for the Timoni provider.
#TimoniProvider: {
	// Install contains whether to install Timoni in the CI environment.
	// +optional
	install: (null | bool) & (_ | *true) @go(Install,*bool)

	// Registries contains the registries to use for publishing Timoni modules
	registries: [...string] @go(Registries,[]string)

	// The version of Timoni to use in CI.
	// +optional
	version: (_ | *"latest") & {
		string
	} @go(Version)
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
