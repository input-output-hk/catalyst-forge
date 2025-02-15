package project

#Deployment: {
    // On contains the events that trigger the deployment.
    on: [string]: _

    // Modules contains the deployment modules for the project.
    modules: #ModuleBundle
}

#ModuleBundle: [string]: #Module

#Module: {
    // Instance contains the instance name to use for all generated resources.
    instance?: string

    // Name contains the name of the module to deploy.
    name?: string

    // Namespace contains the namespace to deploy the module to.
    namespace: string | *"default"

    // Path contains the path to the module.
    path?: string

    // Registry contains the registry to pull the module from.
    registry?: string

    // Type contains the type of the module.
    type: string | *"kcl"

    // Values contains the values to pass to the deployment module.
    values?: _

    // Version contains the version of the deployment module.
    version?: string
}