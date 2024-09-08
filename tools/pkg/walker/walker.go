// Package walker provides implementations for walking over files in a
// filesystem.
package walker

import (
	"io"
	"io/fs"
)

//go:generate go run github.com/matryer/moq@latest -pkg mocks -out mocks/fileseeker.go . FileSeeker
//go:generate go run github.com/matryer/moq@latest -pkg mocks -out mocks/reverse.go . ReverseWalker
//go:generate go run github.com/matryer/moq@latest -pkg mocks -out mocks/walker.go . Walker

// FileType is an enum that represents the type of a file.
type FileType int

const (
	// FileTypeFile represents a file.
	FileTypeFile FileType = iota

	// FileTypeDir represents a directory.
	FileTypeDir
)

// FileSeeker is an interface that combines the fs.File and io.Seeker interfaces.
type FileSeeker interface {
	fs.File
	io.Seeker
}

// WalkerCallback is a callback function that is called for each file in the
// Walk function.
type WalkerCallback func(string, FileType, func() (FileSeeker, error)) error

// Walker is an interface that allows walking over a set of files.
type Walker interface {
	// Walk walks over the files in the given root path and calls the given
	// function for each file.
	Walk(rootPath string, callback WalkerCallback) error
}

// ReverseWalker is an interface that allows reverse walking over a set of
// files.
// The start path is the path where the walk starts and the end path is the path
// where the walk ends.
type ReverseWalker interface {
	// Walk performs a reverse walk from the end path to the start path and
	// calls the given function for each file.
	Walk(startPath, endPath string, callback WalkerCallback) error
}
