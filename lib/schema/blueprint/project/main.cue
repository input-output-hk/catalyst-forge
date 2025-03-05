package project

#Project: {
    // Name contains the name of the project.
    name: string & =~"^[a-z][a-z0-9_-]*$"

    // Container is the name that the container will be built as.
    container: string | *name

    // CI contains the configuration for the CI system.
    ci?: #CI

    // Deployment contains the configuration for the deployment of the project.
    deployment?: #Deployment

    // Release contains the configuration for the release of the project.
    release?: [string]: #Release
}