package billy

import (
	"io/fs"

	"github.com/go-git/go-billy/v5"
)

type BillyFile struct {
	file billy.File
	fs   *BillyFs
}

// Close implements File.Close
func (f *BillyFile) Close() error {
	return f.file.Close()
}

// Name implements File.Name
func (f *BillyFile) Name() string {
	return f.file.Name()
}

// Read implements File.Read
func (f *BillyFile) Read(p []byte) (n int, err error) {
	return f.file.Read(p)
}

// ReadAt implements File.ReadAt
func (f *BillyFile) ReadAt(p []byte, off int64) (n int, err error) {
	return f.file.ReadAt(p, off)
}

// Seek implements File.Seek
func (f *BillyFile) Seek(offset int64, whence int) (int64, error) {
	return f.file.Seek(offset, whence)
}

// Stat implements File.Stat
func (f *BillyFile) Stat() (fs.FileInfo, error) {
	return f.fs.Stat(f.file.Name())
}

// Write implements File.Write
func (f *BillyFile) Write(p []byte) (n int, err error) {
	return f.file.Write(p)
}
