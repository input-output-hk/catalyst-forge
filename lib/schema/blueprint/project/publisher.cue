package project

#Publisher: {
	// Config contains the configuration to pass to the publisher.
	config?: _

	// On contains the events that trigger the publisher.
	on: [string]: _

	// Target is the Earthly target to run for this publisher.
	target: string

	// Type is the type of publisher to use.
	type: string
}
