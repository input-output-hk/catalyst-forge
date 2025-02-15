package global

#Deployment: {
	// Environment contains the default environment to deploy projects to.
	environment: string | *"dev"

	// Registries contains the configuration for the global deployment registries.
	registries: #DeploymentRegistries

	// Repo contains the configuration for the global deployment repository.
	repo: #DeploymentRepo

	// Root contains the root deployment directory in the deployment repository.
	root: string
}

// DeploymentRegistries contains the configuration for the global deployment registries.
#DeploymentRegistries: {
	// Containers contains the default container registry to use for deploying containers.
	containers: string

	// Modules contains the container registry that holds deployment modules.
	modules: string
}

// GlobalDeploymentRepo contains the configuration for the global deployment repository.
#DeploymentRepo: {
	// Ref contains the ref to use for the deployment repository.
	ref: string

	// URL contains the URL of the deployment repository.
	url: string
}
