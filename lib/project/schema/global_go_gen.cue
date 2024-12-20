// Code generated by cue get go. DO NOT EDIT.

//cue:generate cue get go github.com/input-output-hk/catalyst-forge/lib/project/schema

package schema

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

#GlobalRepo: {
	// Name contains the name of the repository (e.g. "owner/repo-name").
	name: string @go(Name)

	// DefaultBranch contains the default branch of the repository.
	defaultBranch: string @go(DefaultBranch)
}
