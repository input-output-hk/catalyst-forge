package global

// Global contains the global configuration for the blueprint.
#Global: {
	// CI contains the configuration for the CI system.
	ci?: #CI

	// Deployment contains the global configuration for the deployment of projects.
	deployment?: #Deployment

	// Deployment contains the global configuration for the deployment of projects.
	repo?: #Repo
}

#Repo: {
	// Name contains the name of the repository (e.g. "owner/repo-name").
	name:          string

	// DefaultBranch contains the default branch of the repository.
	defaultBranch: string
}
