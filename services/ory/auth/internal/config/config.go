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

	Log struct {
		Level     string `mapstructure:"level"`
		Format    string `mapstructure:"format"`
		AddSource bool   `mapstructure:"add_source"`
	} `mapstructure:"log"`
}

// GetServerAddr returns the listen address.
func (c *Config) GetServerAddr() string {
	if c.Server.Addr == "" {
		return ":8080"
	}
	return c.Server.Addr
}

// Validate checks required fields and applies sensible defaults.
func (c *Config) Validate() error {
	if u, err := url.ParseRequestURI(c.Hydra.AdminURL); err != nil || u.Host == "" {
		return errors.New("invalid hydra.admin_url")
	}
	if u, err := url.ParseRequestURI(c.Kratos.PublicURL); err != nil || u.Host == "" {
		return errors.New("invalid kratos.public_url")
	}
	if c.Consent.RememberForSeconds <= 0 {
		c.Consent.RememberForSeconds = 300
	}
	if len(c.Consent.Scopes) == 0 {
		c.Consent.Scopes = []string{"openid", "offline"}
	}
	if c.Log.Level == "" {
		c.Log.Level = "info"
	}
	if c.Log.Format == "" {
		c.Log.Format = "text"
	}
	return nil
}
