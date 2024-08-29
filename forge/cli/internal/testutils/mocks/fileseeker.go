package mocks

import (
	"io/fs"
	"strings"
)

// MockFileSeeker is a mock implementation of the FileSeeker interface.
type MockFileSeeker struct {
	*strings.Reader
}

func (MockFileSeeker) Stat() (fs.FileInfo, error) {
	return MockFileInfo{}, nil
}

func (MockFileSeeker) Close() error {
	return nil
}

// MockFileInfo is a mock implementation of the fs.FileInfo interface.
type MockFileInfo struct {
	fs.FileInfo
}

func (MockFileInfo) Name() string {
	return "Earthfile"
}

// NewMockFileSeeker creates a new MockFileSeeker with the given content.
func NewMockFileSeeker(s string) MockFileSeeker {
	return MockFileSeeker{strings.NewReader(s)}
}
