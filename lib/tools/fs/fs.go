package fs

import (
	"os"
	"path/filepath"
)

type Filesystem interface {
	Create(name string) (File, error)
	Exists(path string) (bool, error)
	MkdirAll(path string, perm os.FileMode) error
	Open(name string) (File, error)
	OpenFile(name string, flag int, perm os.FileMode) (File, error)
	ReadDir(dirname string) ([]os.FileInfo, error)
	ReadFile(path string) ([]byte, error)
	Remove(name string) error
	Stat(name string) (os.FileInfo, error)
	TempDir(dir string, prefix string) (name string, err error)
	Walk(root string, walkFn filepath.WalkFunc) error
	WriteFile(filename string, data []byte, perm os.FileMode) error
}
