package schema

// Deployment contains the configuration for the deployment of the project.
type Deployment struct {
	// On contains the events that trigger the deployment.
	On map[string]any `json:"on"`

	// Modules contains the deployment modules for the project.
	Modules map[string]DeploymentModule `json:"modules"`
}

// Module contains the configuration for a deployment module.
type DeploymentModule struct {
	// Name contains the name of the module to deploy.
	// +optional
	Name string `json:"name"`

	// Namespace contains the namespace to deploy the module to.
	Namespace string `json:"namespace"`

	// Values contains the values to pass to the deployment module.
	Values any `json:"values"`

	// Version contains the version of the deployment module.
	Version string `json:"version"`
}
