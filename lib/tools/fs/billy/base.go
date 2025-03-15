package billy

import (
	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/osfs"
)

// BillyBaseOsFs is a billy.Filesystem that acts like the native filesystem.
type BillyBaseOsFs struct {
	osfs.ChrootOS
}

func (b *BillyBaseOsFs) Chroot(path string) (billy.Filesystem, error) {
	return osfs.New(path), nil
}

func (b *BillyBaseOsFs) Root() string {
	return "/"
}

// NewBaseOsFS creates a new OS filesystem that acts like the native filesystem.
func NewBaseOsFS() *BillyFs {
	return &BillyFs{
		fs: &BillyBaseOsFs{},
	}
}
