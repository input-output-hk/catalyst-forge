package project

#Release: {
    // Config contains the configuration to pass to the release.
    config?: _

    // On contains the events that trigger the release.
    on: [string]: _

    // Target is the Earthly target to run for this release.
	// Defaults to release name.
    target?: string
}