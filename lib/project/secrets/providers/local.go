package providers

import (
	"log/slog"

	"github.com/input-output-hk/catalyst-forge/lib/tools/fs"
	"github.com/input-output-hk/catalyst-forge/lib/tools/fs/billy"
)

type LocalClient struct {
	fs     fs.Filesystem
	logger *slog.Logger
}

func NewLocalClient(logger *slog.Logger) (*LocalClient, error) {
	return &LocalClient{
		fs:     billy.NewBaseOsFS(),
		logger: logger,
	}, nil
}

func (c *LocalClient) Get(key string) (string, error) {
	b, err := c.fs.ReadFile(key)
	if err != nil {
		return string(b), err
	}

	return string(b), nil
}

func (c *LocalClient) Set(key, value string) (string, error) {
	panic("not implemented")
}
