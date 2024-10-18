// Code generated by cue get go. DO NOT EDIT.

//cue:generate cue get go github.com/input-output-hk/catalyst-forge/lib/project/schema

package schema

// Project contains the configuration for the project.
#Project: {
	// Name contains the name of the project.
	name: string @go(Name)

	// Container is the name that the container will be built as.
	container: string @go(Container)

	// CI contains the configuration for the CI system.
	// +optional
	ci?: #ProjectCI @go(CI)

	// Deployment contains the configuration for the deployment of the project.
	// +optional
	deployment?: #Deployment @go(Deployment)

	// Release contains the configuration for the release of the project.
	// +optional
	release?: {[string]: #Release} @go(Release,map[string]Release)
}

#ProjectCI: {
	// Targets configures the individual targets that are run by the CI system.
	// +optional
	targets?: {[string]: #Target} @go(Targets,map[string]Target)
}

// Release contains the configuration for a project release.
#Release: {
	// Config contains the configuration to pass to the release.
	// +optional
	config?: _ @go(Config,any)

	// On contains the events that trigger the release.
	on: {...} @go(On,map[string]any)

	// Target is the Earthly target to run for this release.
	// Defaults to release name.
	// +optional
	target?: string @go(Target)
}

// Target contains the configuration for a single target.
#Target: {
	// Args contains the arguments to pass to the target.
	// +optional
	args?: {[string]: string} @go(Args,map[string]string)

	// Platforms contains the platforms to run the target against.
	// +optional
	platforms?: [...string] @go(Platforms,[]string)

	// Privileged determines if the target should run in privileged mode.
	// +optional
	privileged?: null | bool @go(Privileged,*bool)

	// Retries contains the number of times to retry the target.
	// +optional
	retries?: null | int @go(Retries,*int)

	// Secrets contains the secrets to pass to the target.
	// +optional
	secrets?: [...#Secret] @go(Secrets,[]Secret)
}
