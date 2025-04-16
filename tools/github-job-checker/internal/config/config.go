package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// Config holds the configuration values for the application.
type Config struct {
	Owner         string        `mapstructure:"owner"`
	Repo          string        `mapstructure:"repo"`
	Ref           string        `mapstructure:"ref"`
	Token         string        `mapstructure:"token"`
	CheckInterval time.Duration `mapstructure:"check_interval"`
	Timeout       time.Duration `mapstructure:"timeout"`
}

// LoadConfig loads the configuration from flags, environment variables, or a config file.
func LoadConfig() (*Config, error) {
	// Set up flag normalization to replace hyphens with underscores
	pflag.CommandLine.SetNormalizeFunc(func(f *pflag.FlagSet, name string) pflag.NormalizedName {
		return pflag.NormalizedName(strings.ReplaceAll(name, "-", "_"))
	})

	// Define command-line flags.
	pflag.String("owner", "", "Repository owner")
	pflag.String("repo", "", "Repository name")
	pflag.String("ref", "", "Commit hash or reference")
	pflag.String("token", "", "GitHub token")
	pflag.Duration("check-interval", 10*time.Second, "Interval between checks")
	pflag.Duration("timeout", 300*time.Second, "Timeout for the operation")

	// Parse the command-line flags.
	pflag.Parse()

	// Bind the command-line flags to Viper.
	if err := viper.BindPFlags(pflag.CommandLine); err != nil {
		return nil, fmt.Errorf("failed to bind flags: %w", err)
	}

	// Set environment variable prefixes.
	viper.SetEnvPrefix("GHJC")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()

	//  // Check if a config file is already set.
	if viper.ConfigFileUsed() == "" {
		// Optionally read from a config file.
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(".")
	}

	if err := viper.ReadInConfig(); err != nil {
		// It's okay if the config file doesn't exist.
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	// Unmarshal the config into the Config struct.
	var cfg Config
	if err := viper.Unmarshal(&cfg, viper.DecodeHook(
		mapstructure.StringToTimeDurationHookFunc(),
	)); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	fmt.Printf("All Settings: %+v\n", viper.AllSettings())

	// Validate required fields.
	if cfg.Owner == "" {
		return nil, fmt.Errorf("owner is required")
	}
	if cfg.Repo == "" {
		return nil, fmt.Errorf("repo is required")
	}
	if cfg.Ref == "" {
		return nil, fmt.Errorf("ref is required")
	}
	if cfg.Token == "" {
		return nil, fmt.Errorf("token is required")
	}

	return &cfg, nil
}
