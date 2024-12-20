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

	// Release contains the configuration for the release of the project.
	// +optional
	Release map[string]Release `json:"release"`
}

type ProjectCI struct {
	// Targets configures the individual targets that are run by the CI system.
	// +optional
	Targets map[string]Target `json:"targets"`
}

// Release contains the configuration for a project release.
type Release struct {
	// Config contains the configuration to pass to the release.
	// +optional
	Config any `json:"config"`

	// On contains the events that trigger the release.
	On map[string]any `json:"on"`

	// Target is the Earthly target to run for this release.
	// Defaults to release name.
	// +optional
	Target string `json:"target"`
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
