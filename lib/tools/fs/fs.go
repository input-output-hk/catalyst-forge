package fs

import "os"

type Filesystem interface {
	MkdirAll(path string, perm os.FileMode) error
	ReadDir(dirname string) ([]os.FileInfo, error)
	ReadFile(path string) ([]byte, error)
	Remove(name string) error
	Stat(name string) (os.FileInfo, error)
	WriteFile(filename string, data []byte, perm os.FileMode) error
}
