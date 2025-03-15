package fs

import "io/fs"

type File interface {
	Close() error
	Name() string
	Read(p []byte) (n int, err error)
	ReadAt(p []byte, off int64) (n int, err error)
	Seek(offset int64, whence int) (int64, error)
	Stat() (fs.FileInfo, error)
	Write(p []byte) (n int, err error)
}
