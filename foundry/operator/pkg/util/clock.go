package util

import (
	"time"
)

// Clock provides an interface for time operations to allow easy mocking in tests.
type Clock interface {
	// Now returns the current time.
	Now() time.Time

	// Since returns the duration elapsed since the given time.
	Since(t time.Time) time.Duration

	// Until returns the duration until the given time.
	Until(t time.Time) time.Duration
}

// RealClock is a Clock that uses the time package functions.
type RealClock struct{}

func (RealClock) Now() time.Time {
	return time.Now()
}

func (RealClock) Since(t time.Time) time.Duration {
	return time.Since(t)
}

func (RealClock) Until(t time.Time) time.Duration {
	return time.Until(t)
}
