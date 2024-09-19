package providers

import (
	"fmt"
	"log/slog"
	"os"
)

type EnvClient struct {
	logger *slog.Logger
}

func NewEnvClient(logger *slog.Logger) (*EnvClient, error) {
	return &EnvClient{
		logger: logger,
	}, nil
}

func (c *EnvClient) Get(key string) (string, error) {
	c.logger.Debug("Getting secret from environment variable", "key", key)

	secret, ok := os.LookupEnv(key)
	if !ok {
		return "", fmt.Errorf("enviroment variable %s not found", key)
	}

	return secret, nil
}

func (c *EnvClient) Set(key, value string) (string, error) {
	panic("not implemented")
}
