package schema

// Project contains the configuration for the project.
type Project struct {
	// Name contains the name of the project.
	Name string `json:"name"`

	// Container is the name that the container will be built as.
	Container string `json:"container"`

	// CI contains the configuration for the CI system.
	// +optional
	CI ProjectCI `json:"ci"`

	// Deployment contains the configuration for the deployment of the project.
	// +optional
	Deployment Deployment `json:"deployment"`
}

type ProjectCI struct {
	// Targets configures the individual targets that are run by the CI system.
	// +optional
	Targets map[string]Target `json:"targets"`
}

// Target contains the configuration for a single target.
type Target struct {
	// Args contains the arguments to pass to the target.
	// +optional
	Args map[string]string `json:"args"`

	// Platforms contains the platforms to run the target against.
	// +optional
	Platforms []string `json:"platforms"`

	// Privileged determines if the target should run in privileged mode.
	// +optional
	Privileged *bool `json:"privileged"`

	// Retries contains the number of times to retry the target.
	// +optional
	Retries *int `json:"retries"`

	// Secrets contains the secrets to pass to the target.
	// +optional
	Secrets []Secret `json:"secrets"`
}
