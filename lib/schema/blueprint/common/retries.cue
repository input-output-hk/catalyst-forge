package common

// CIRetries contains the configuration for the retries of an Earthly target.
#CIRetries: {
	// Attempts contains the number of attempts to retry the target.
	attempts: int | *0

	// Delay contains the delay between retries.
	delay: string | *"10s"

	// Filters filters retries based on the log lines from the Earthly run.
	filters?: [...string]
}
