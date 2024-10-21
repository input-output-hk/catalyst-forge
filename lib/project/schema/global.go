package schema

// Global contains the global configuration for the blueprint.
type Global struct {
	// CI contains the configuration for the CI system.
	// +optional
	CI GlobalCI `json:"ci"`

	// Deployment contains the global configuration for the deployment of projects.
	// +optional
	Deployment GlobalDeployment `json:"deployment"`

	// Repo contains the configuration for the GitHub repository.
	Repo GlobalRepo `json:"repo"`
}

// CI contains the configuration for the CI system.
type GlobalCI struct {
	// Local defines the filters to use when simulating a local CI run.
	Local []string `json:"local"`

	// Providers contains the configuration for the providers being used by the CI system.
	// +optional
	Providers Providers `json:"providers"`

	// Registries contains the container registries to push images to.
	// +optional
	Registries []string `json:"registries"`

	// Secrets contains global secrets that will be passed to all targets.
	// +optional
	Secrets []Secret `json:"secrets"`
}

// GlobalDeployment contains the configuration for the global deployment of projects.
type GlobalDeployment struct {
	// Registry contains the URL of the container registry holding the deployment code.
	Registry string `json:"registry"`

	// Repo contains the configuration for the global deployment repository.
	Repo GlobalDeploymentRepo `json:"repo"`

	// Root contains the root deployment directory in the deployment repository.
	Root string `json:"root"`
}

// GlobalDeploymentRepo contains the configuration for the global deployment repository.
type GlobalDeploymentRepo struct {
	// Ref contains the ref to use for the deployment repository.
	Ref string `json:"ref"`

	// URL contains the URL of the deployment repository.
	Url string `json:"url"`
}

type GlobalRepo struct {
	// Name contains the name of the repository (e.g. "owner/repo-name").
	Name string `json:"name"`

	// DefaultBranch contains the default branch of the repository.
	DefaultBranch string `json:"defaultBranch"`
}
