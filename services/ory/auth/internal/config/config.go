package config

import (
	"errors"
	"net/url"
)

// Config holds service configuration.
type Config struct {
	Server struct {
		Addr string `mapstructure:"addr"`
	} `mapstructure:"server"`

	Hydra struct {
		AdminURL string `mapstructure:"admin_url"`
	} `mapstructure:"hydra"`

	Kratos struct {
		PublicURL string `mapstructure:"public_url"`
	} `mapstructure:"kratos"`

	Consent struct {
		RememberForSeconds int      `mapstructure:"remember_for"`
		Scopes             []string `mapstructure:"scopes"`
		Audience           []string `mapstructure:"audience"`
	} `mapstructure:"consent"`

	TLS struct {
		CAFile string `mapstructure:"ca_file"`
		CAPem  string `mapstructure:"ca_pem"`
	} `mapstructure:"tls"`

	Mapping struct {
		Path          string `mapstructure:"path"`
		OnError       string `mapstructure:"on_error"`
		MergeStrategy string `mapstructure:"merge_strategy"`
		Reload        bool   `mapstructure:"reload"`
	} `mapstructure:"mapping"`

	Log struct {
		Level     string `mapstructure:"level"`
		Format    string `mapstructure:"format"`
		AddSource bool   `mapstructure:"add_source"`
	} `mapstructure:"log"`
}

// GetServerAddr returns the listen address.
func (c *Config) GetServerAddr() string {
	return c.Server.Addr
}

// Validate checks required fields.
func (c *Config) Validate() error {
	if u, err := url.ParseRequestURI(c.Hydra.AdminURL); err != nil || u.Host == "" {
		return errors.New("invalid hydra.admin_url")
	}
	if u, err := url.ParseRequestURI(c.Kratos.PublicURL); err != nil || u.Host == "" {
		return errors.New("invalid kratos.public_url")
	}
	return nil
}
