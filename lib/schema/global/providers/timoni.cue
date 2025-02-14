package providers

#Timoni: {
    // Install contains whether to install Timoni in the CI environment.
    install?: bool | *false

    // Registries contains the registries to use for publishing Timoni modules
    registries: [...string]

    // Version contains the version of Timoni to install in CI.
    version?: string
}