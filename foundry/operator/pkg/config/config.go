package config

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/input-output-hk/catalyst-forge/lib/deployment/deployer"
)

// OperatorConfig is the configuration for the operator.
type OperatorConfig struct {
	Api         APIConfig               `json:"api"`
	Deployer    deployer.DeployerConfig `json:"deployer"`
	MaxAttempts int                     `json:"max_attempts"`
}

type APIConfig struct {
	Url       string `json:"url"`
	TokenPath string `json:"token_path"`
}

// Load loads the operator configuration from the given path.
func Load(path string) (OperatorConfig, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return OperatorConfig{}, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg OperatorConfig
	if err := json.Unmarshal(contents, &cfg); err != nil {
		return OperatorConfig{}, fmt.Errorf("failed to unmarshal config file: %w", err)
	}

	return cfg, nil
}
