package schema

type DeploymentModuleBundle map[string]DeploymentModule

// Deployment contains the configuration for the deployment of the project.
type Deployment struct {
	// On contains the events that trigger the deployment.
	On map[string]any `json:"on"`

	// Modules contains the deployment modules for the project.
	Modules DeploymentModuleBundle `json:"modules"`
}

// Module contains the configuration for a deployment module.
type DeploymentModule struct {
	// Environment contains the environment the module is being deployed to.
	// This value should never be set by the user. It is set by the system.
	// +optional
	Environment *string `json:"environment"`

	// Instance contains the instance name to use for all generated resources.
	// +optional
	Instance string `json:"instance"`

	// Name contains the name of the module to deploy.
	// +optional
	Name *string `json:"name"`

	// Namespace contains the namespace to deploy the module to.
	Namespace string `json:"namespace"`

	// Path contains the path to the module.
	// +optional
	Path *string `json:"path"`

	// Registry contains the registry to pull the module from.
	// +optional
	Registry *string `json:"registry"`

	// Type contains the type of the module.
	Type string `json:"type"`

	// Values contains the values to pass to the deployment module.
	Values any `json:"values"`

	// Version contains the version of the deployment module.
	// +optional
	Version *string `json:"version"`
}
