package providers

import (
	"log/slog"

	"github.com/spf13/afero"
)

type LocalClient struct {
	fs     afero.Fs
	logger *slog.Logger
}

func NewLocalClient(logger *slog.Logger) (*LocalClient, error) {
	return &LocalClient{
		fs:     afero.NewOsFs(),
		logger: logger,
	}, nil
}

func (c *LocalClient) Get(key string) (string, error) {
	b, err := afero.ReadFile(c.fs, key)
	if err != nil {
		return string(b), err
	}

	return string(b), nil
}

func (c *LocalClient) Set(key, value string) (string, error) {
	panic("not implemented")
}
