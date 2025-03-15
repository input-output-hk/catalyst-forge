package billy

import (
	"os"
	"path/filepath"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-billy/v5/util"
	"github.com/input-output-hk/catalyst-forge/lib/tools/fs"
)

// BillyFs implements the Filesystem interface using go-billy
type BillyFs struct {
	fs billy.Filesystem
}

// Create implements Filesystem.Create
func (b *BillyFs) Create(name string) (fs.File, error) {
	f, err := b.fs.Create(name)
	if err != nil {
		return nil, err
	}
	return &BillyFile{
		file: f,
		fs:   b,
	}, nil
}

// Exists implements Filesystem.Exists
func (b *BillyFs) Exists(path string) (bool, error) {
	_, err := b.fs.Stat(path)
	if err == nil {
		return true, nil
	} else if os.IsNotExist(err) {
		return false, nil
	} else {
		return false, err
	}
}

// MkdirAll implements Filesystem.MkdirAll
func (b *BillyFs) MkdirAll(path string, perm os.FileMode) error {
	return b.fs.MkdirAll(path, perm)
}

// Open implements Filesystem.Open
func (b *BillyFs) Open(name string) (fs.File, error) {
	f, err := b.fs.Open(name)
	if err != nil {
		return nil, err
	}
	return &BillyFile{
		file: f,
		fs:   b,
	}, nil
}

// OpenFile implements Filesystem.OpenFile
func (b *BillyFs) OpenFile(name string, flag int, perm os.FileMode) (fs.File, error) {
	f, err := b.fs.OpenFile(name, flag, perm)
	if err != nil {
		return nil, err
	}
	return &BillyFile{
		file: f,
		fs:   b,
	}, nil
}

// ReadDir implements Filesystem.ReadDir
func (b *BillyFs) ReadDir(dirname string) ([]os.FileInfo, error) {
	return b.fs.ReadDir(dirname)
}

// ReadFile implements Filesystem.ReadFile
func (b *BillyFs) ReadFile(path string) ([]byte, error) {
	return util.ReadFile(b.fs, path)
}

// Remove implements Filesystem.Remove
func (b *BillyFs) Remove(name string) error {
	return b.fs.Remove(name)
}

// Stat implements Filesystem.Stat
func (b *BillyFs) Stat(name string) (os.FileInfo, error) {
	return b.fs.Stat(name)
}

// TempDir implements Filesystem.TempDir
func (b *BillyFs) TempDir(dir string, prefix string) (name string, err error) {
	return util.TempDir(b.fs, dir, prefix)
}

// Walk implements Filesystem.Walk
func (b *BillyFs) Walk(root string, walkFn filepath.WalkFunc) error {
	return util.Walk(b.fs, root, walkFn)
}

// WriteFile implements Filesystem.WriteFile
func (b *BillyFs) WriteFile(filename string, data []byte, perm os.FileMode) error {
	return util.WriteFile(b.fs, filename, data, perm)
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
