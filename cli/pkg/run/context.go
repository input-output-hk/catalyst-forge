package run

// RunContext represents the context in which a CLI run is happening.
type RunContext struct {
	// CI is true if the run is happening in a CI environment.
	CI bool

	// Local is true if the run is happening in a local environment.
	Local bool

	// Verbose is the verbosity level of the run.
	Verbose int
}
