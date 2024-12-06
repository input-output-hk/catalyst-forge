package schema

// Deployment contains the configuration for the deployment of the project.
type Deployment struct {
	// Environment contains the environment to deploy the module to.
	Environment string `json:"environment"`

	// On contains the events that trigger the deployment.
	On map[string]any `json:"on"`

	// Modules contains the configuration for the deployment modules for the project.
	// +optional
	Modules *DeploymentModules `json:"modules"`
}

// Deployment contains the configuration for the deployment of the project.
type DeploymentModules struct {
	// Main contains the configuration for the main deployment module.
	Main Module `json:"main"`

	// Support contains the configuration for the support deployment modules.
	// +optional
	Support map[string]Module `json:"support"`
}

// Module contains the configuration for a deployment module.
type Module struct {
	// Container contains the name of the container holding the deployment code.
	// Defaults to <module_name>-deployment). For the main module, <module_name> is the project name.
	// +optional
	Container *string `json:"container"`

	// Namespace contains the namespace to deploy the module to.
	Namespace string `json:"namespace"`

	// Values contains the values to pass to the deployment module.
	Values any `json:"values"`

	// Version contains the version of the deployment module.
	Version string `json:"version"`
}
