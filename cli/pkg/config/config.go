package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/input-output-hk/catalyst-forge/lib/tools/fs"
	"github.com/input-output-hk/catalyst-forge/lib/tools/fs/billy"
)

const (
	ConfigFilename = "config.toml"
)

type CLIConfig struct {
	Email string `toml:"email"`
	Token string `toml:"token"`
	fs    fs.Filesystem
}

// Save saves the CLIConfig to the default config file.
func (c *CLIConfig) Save() error {
	configPath, err := c.ConfigPath()
	if err != nil {
		return err
	}

	if err := c.fs.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := c.ToBytes()
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := c.fs.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file %s: %w", configPath, err)
	}

	return nil
}

// ToBytes converts a CLIConfig to TOML bytes
func (c *CLIConfig) ToBytes() ([]byte, error) {
	return toml.Marshal(c)
}

// ToString converts a CLIConfig to a TOML string
func (c *CLIConfig) ToString() (string, error) {
	data, err := c.ToBytes()
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// Load loads the CLIConfig from the default config file.
func (c *CLIConfig) Load() error {
	configPath, err := c.ConfigPath()
	if err != nil {
		return err
	}

	data, err := c.fs.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file %s: %w", configPath, err)
	}

	if err := toml.Unmarshal(data, c); err != nil {
		return fmt.Errorf("failed to parse TOML: %w", err)
	}

	return nil
}

// Exists checks if the config file exists.
func (c *CLIConfig) Exists() (bool, error) {
	configPath, err := c.ConfigPath()
	if err != nil {
		return false, err
	}

	_, err = os.Stat(configPath)
	if err != nil {
		return false, nil
	}

	return true, nil
}

// ConfigPath returns the path to the config file.
func (c *CLIConfig) ConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	return filepath.Join(home, ".config", "forge", ConfigFilename), nil
}

// NewConfig creates a new CLIConfig with the default filesystem.
func NewConfig() *CLIConfig {
	return &CLIConfig{
		fs: billy.NewBaseOsFS(),
	}
}

// NewCustomConfig creates a new CLIConfig with a custom filesystem.
func NewCustomConfig(fs fs.Filesystem) *CLIConfig {
	return &CLIConfig{
		fs: fs,
	}
}
