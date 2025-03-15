package fs

import (
	"os"

	"github.com/spf13/afero"
)

// AferoFs implements the Filesystem interface using afero
type AferoFs struct {
	fs afero.Fs
}

// MkdirAll implements Filesystem.MkdirAll
func (a *AferoFs) MkdirAll(path string, perm os.FileMode) error {
	return a.fs.MkdirAll(path, perm)
}

// ReadDir implements Filesystem.ReadDir
func (a *AferoFs) ReadDir(dirname string) ([]os.FileInfo, error) {
	return afero.ReadDir(a.fs, dirname)
}

// ReadFile implements Filesystem.ReadFile
func (a *AferoFs) ReadFile(path string) ([]byte, error) {
	return afero.ReadFile(a.fs, path)
}

// Remove implements Filesystem.Remove
func (a *AferoFs) Remove(name string) error {
	return a.fs.Remove(name)
}

// Stat implements Filesystem.Stat
func (a *AferoFs) Stat(name string) (os.FileInfo, error) {
	return a.fs.Stat(name)
}

// WriteFile implements Filesystem.WriteFile
func (a *AferoFs) WriteFile(filename string, data []byte, perm os.FileMode) error {
	return afero.WriteFile(a.fs, filename, data, perm)
}

// Raw returns the underlying afero filesystem.
func (f *AferoFs) Raw() afero.Fs {
	return f.fs
}

// NewFs creates a new filesystem using the given afero filesystem.
func NewFs(fs afero.Fs) *AferoFs {
	return &AferoFs{
		fs: fs,
	}
}

// NewInMemoryFs creates a new in-memory filesystem.
func NewInMemoryFs() *AferoFs {
	return &AferoFs{
		fs: afero.NewMemMapFs(),
	}
}

// NewOsFs creates a new OS filesystem.
func NewOsFs() *AferoFs {
	return &AferoFs{
		fs: afero.NewOsFs(),
	}
}
