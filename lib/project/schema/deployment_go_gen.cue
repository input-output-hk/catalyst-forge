// Code generated by cue get go. DO NOT EDIT.

//cue:generate cue get go github.com/input-output-hk/catalyst-forge/lib/project/schema

package schema

// Deployment contains the configuration for the deployment of the project.
#Deployment: {
	// Environment contains the environment to deploy the module to.
	environment: string @go(Environment)

	// Modules contains the configuration for the deployment modules for the project.
	// +optional
	modules?: null | #DeploymentModules @go(Modules,*DeploymentModules)
}

// Deployment contains the configuration for the deployment of the project.
#DeploymentModules: {
	// Main contains the configuration for the main deployment module.
	main: #Module @go(Main)

	// Support contains the configuration for the support deployment modules.
	// +optional
	support?: {[string]: #Module} @go(Support,map[string]Module)
}

// Module contains the configuration for a deployment module.
#Module: {
	// Container contains the name of the container holding the deployment code.
	// Defaults to <module_name>-deployment). For the main module, <module_name> is the project name.
	// +optional
	container?: null | string @go(Container,*string)

	// Namespace contains the namespace to deploy the module to.
	namespace: string @go(Namespace)

	// Values contains the values to pass to the deployment module.
	values: _ @go(Values,any)

	// Version contains the version of the deployment module.
	version: string @go(Version)
}
