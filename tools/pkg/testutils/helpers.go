// Package testutils provides a set of helper functions for testing.
package testutils

import (
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

// AssertError asserts the state of an error.
//
// If isExpected is true, the function will assert that the error is not nil.
// If expectedErr is not empty, the function will assert that the error message
// matches the expected error message.
// If isExpected is false, the function will assert that the error is nil.
// The function returns true if an error was found, false otherwise.
func AssertError(t *testing.T, err error, isExpected bool, expectedErr string) bool {
	if isExpected {
		require.Error(t, err)

		if expectedErr != "" {
			require.EqualError(t, err, expectedErr)
		}

		return true
	} else {
		require.NoError(t, err)
	}

	return false
}

// SetupFS sets up the filesystem with the given files.
func SetupFS(t *testing.T, fs afero.Fs, files map[string]string) {
	for path, content := range files {
		dir := filepath.Dir(path)
		if err := fs.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("failed to create directory %s: %v", dir, err)
		}

		if err := afero.WriteFile(fs, path, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write file %s: %v", path, err)
		}
	}
}
