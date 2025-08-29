package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// FileConfig defines the YAML-driven, multi-issuer configuration.
type FileConfig struct {
	ListenAddr     string           `yaml:"listen_addr"`
	BaseURL        string           `yaml:"base_url"`
	GlobalSecret   string           `yaml:"global_secret"`
	AllowPKCEPlain bool             `yaml:"allow_pkce_plain"`
	Providers      []ProviderConfig `yaml:"providers"`
}

type ProviderConfig struct {
	ID                       string                   `yaml:"id"`
	Public                   bool                     `yaml:"public"`
	Client                   ProviderClientConfig     `yaml:"client"`
	Personas                 map[string]PersonaConfig `yaml:"personas"`
	ExtraIDTokenClaims       map[string]any           `yaml:"extra_id_token_claims"`
	UserinfoClaimsFromScopes map[string][]string      `yaml:"userinfo_claims_from_scopes"`
	SigningKeyPEM            string                   `yaml:"signing_key_pem"`
	SigningKeyPEMPath        string                   `yaml:"signing_key_pem_path"`
	AllowPKCEPlain           *bool                    `yaml:"allow_pkce_plain"`
	BaseURL                  string                   `yaml:"base_url"`
	Override                 bool                     `yaml:"override"`
	IssuerOverride           string                   `yaml:"issuer_override"`
	DefaultAudience          []string                 `yaml:"default_audience"`
}

type ProviderClientConfig struct {
	ID           string   `yaml:"id"`
	Secret       string   `yaml:"secret"`
	RedirectURIs []string `yaml:"redirect_uris"`
}

type PersonaConfig struct {
	Sub    string         `yaml:"sub"`
	Claims map[string]any `yaml:"claims"`
}

// LoadFromEnv loads the YAML config from CONFIG_PATH.
func LoadFromEnv() (*FileConfig, error) {
	path := os.Getenv("CONFIG_PATH")
	if strings.TrimSpace(path) == "" {
		return nil, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read CONFIG_PATH: %w", err)
	}
	var cfg FileConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse YAML: %w", err)
	}
	if cfg.BaseURL != "" {
		cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/")
	}
	if len([]byte(cfg.GlobalSecret)) != 32 {
		return nil, fmt.Errorf("global_secret must be exactly 32 bytes (got %d)", len([]byte(cfg.GlobalSecret)))
	}
	return &cfg, nil
}
