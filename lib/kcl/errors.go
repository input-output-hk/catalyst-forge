package kcl

import (
	"fmt"
)

// WrapError wraps an error with additional context.
func WrapError(err error, format string, args ...interface{}) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", fmt.Sprintf(format, args...), err)
}

// ErrorWithDigest adds digest information to an error.
func ErrorWithDigest(err error, digest string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w (digest: %s)", err, digest)
}

// ErrorWithProfile adds profile information to an error.
func ErrorWithProfile(err error, profile Profile) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w (profile: %s)", err, profile)
}

// ErrorWithIntent adds intent hash information to an error.
func ErrorWithIntent(err error, intentHash string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w (intent: %s)", err, intentHash)
}

// IsRetryable returns true if the error is transient and operation can be retried.
func IsRetryable(err error) bool {
	// Check for specific retryable conditions
	// This can be expanded based on actual error types from OCI client
	return false
}

// IsCacheError returns true if the error is cache-related.
func IsCacheError(err error) bool {
	return err == ErrCacheCorrupt
}

// ValidationError represents a validation failure.
type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("validation failed for %s: %s", e.Field, e.Message)
}

// MultiError represents multiple errors.
type MultiError struct {
	Errors []error
}

func (e MultiError) Error() string {
	if len(e.Errors) == 0 {
		return "no errors"
	}
	if len(e.Errors) == 1 {
		return e.Errors[0].Error()
	}
	return fmt.Sprintf("%d errors occurred: first error: %s", len(e.Errors), e.Errors[0])
}

// Add adds an error to the multi-error.
func (e *MultiError) Add(err error) {
	if err != nil {
		e.Errors = append(e.Errors, err)
	}
}

// HasErrors returns true if there are any errors.
func (e *MultiError) HasErrors() bool {
	return len(e.Errors) > 0
}

// Err returns the multi-error or nil if no errors.
func (e *MultiError) Err() error {
	if !e.HasErrors() {
		return nil
	}
	return e
}