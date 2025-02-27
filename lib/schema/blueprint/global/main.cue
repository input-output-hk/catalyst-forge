package global

// Global contains the global configuration for the blueprint.
#Global: {
	// CI contains the configuration for the CI system.
	ci?: #CI

	// Deployment contains the global configuration for the deployment of projects.
	deployment?: #Deployment

	// Deployment contains the global configuration for the deployment of projects.
	repo?: #Repo

	// State is an optional field that can be used to store global state for later use.
	// This can be used by external tools or can be consumed using the @global() attribute.
	// This field is not used by the blueprint itself.
	state?: _
}

#Repo: {
	// Name contains the name of the repository (e.g. "owner/repo-name").
	name:          string

	// DefaultBranch contains the default branch of the repository.
	defaultBranch: string
}
