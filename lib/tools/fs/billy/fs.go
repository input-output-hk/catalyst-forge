package fs

import (
	"io"
	"os"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-billy/v5/osfs"
)

// BillyFs implements the Filesystem interface using go-billy
type BillyFs struct {
	fs billy.Filesystem
}

// MkdirAll implements Filesystem.MkdirAll
func (b *BillyFs) MkdirAll(path string, perm os.FileMode) error {
	return b.fs.MkdirAll(path, perm)
}

// ReadDir implements Filesystem.ReadDir
func (b *BillyFs) ReadDir(dirname string) ([]os.FileInfo, error) {
	return b.fs.ReadDir(dirname)
}

// ReadFile implements Filesystem.ReadFile
func (b *BillyFs) ReadFile(path string) ([]byte, error) {
	file, err := b.fs.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return io.ReadAll(file)
}

// Remove implements Filesystem.Remove
func (b *BillyFs) Remove(name string) error {
	return b.fs.Remove(name)
}

// Stat implements Filesystem.Stat
func (b *BillyFs) Stat(name string) (os.FileInfo, error) {
	return b.fs.Stat(name)
}

// WriteFile implements Filesystem.WriteFile
func (b *BillyFs) WriteFile(filename string, data []byte, perm os.FileMode) error {
	file, err := b.fs.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(data)
	return err
}

// Raw returns the underlying go-billy filesystem.
func (b *BillyFs) Raw() billy.Filesystem {
	return b.fs
}

// NewFs creates a new BillyFs using the given go-billy filesystem.
func NewFs(fs billy.Filesystem) *BillyFs {
	return &BillyFs{
		fs: fs,
	}
}

// NewInMemoryFs creates a new in-memory filesystem.
func NewInMemoryFs() *BillyFs {
	return &BillyFs{
		fs: memfs.New(),
	}
}

// NewOsFs creates a new OS filesystem.
func NewOsFs(path string) *BillyFs {
	return &BillyFs{
		fs: osfs.New(path),
	}
}
