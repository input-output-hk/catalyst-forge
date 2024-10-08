package testutils

import (
	"fmt"
	"io"
	"log/slog"
	"testing"
)

// CheckError checks if the error is expected or not. If the error is
// unexpected, it returns an error.
// If the function returns true, the test should return immediately.
func CheckError(t *testing.T, err error, expected bool, expectedErr error) (bool, error) {
	if expected && err != nil {
		fmt.Println(expectedErr)
		if expectedErr != nil && err.Error() != expectedErr.Error() {
			return true, fmt.Errorf("got error %v, want error %v", err, expectedErr)
		}
		return true, nil
	} else if !expected && err != nil {
		return true, fmt.Errorf("unexpected error: %v", err)
	} else if expected && err == nil {
		return true, fmt.Errorf("expected error %v, got nil", expectedErr)
	}

	return false, nil
}

// NewNoopLogger creates a new logger that discards all logs.
func NewNoopLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
