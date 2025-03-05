package providers

#KCL: {
    // Install contains whether to install KCL in the CI environment.
    install?: bool | *false

    // Registries contains the registries to use for publishing KCL modules
    registries: [...string]

    // Version contains the version of KCL to install in CI.
    version?: string
}