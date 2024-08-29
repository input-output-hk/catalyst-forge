package walker

import (
	"io"
	"io/fs"
)

//go:generate go run github.com/matryer/moq@latest -out walker_mock.go . Walker

// FileType is an enum that represents the type of a file.
type FileType int

const (
	FileTypeFile FileType = iota
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
