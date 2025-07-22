package satellite

import (
	"encoding/base64"
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/input-output-hk/catalyst-forge/lib/project/project"
	"github.com/input-output-hk/catalyst-forge/lib/project/providers"
	"github.com/input-output-hk/catalyst-forge/lib/project/secrets"
	"github.com/input-output-hk/catalyst-forge/lib/tools/fs"
	"github.com/input-output-hk/catalyst-forge/lib/tools/fs/billy"
	"gopkg.in/yaml.v3"
)

// EarthlySatellite is used to configure the local system to use a remote Earthly Satellite.
type EarthlySatellite struct {
	ci          bool
	creds       providers.EarthlyProviderCreds
	fs          fs.Filesystem
	logger      *slog.Logger
	path        string
	project     *project.Project
	secretStore secrets.SecretStore
}

// EarthlyConfig is the configuration for Earthly.
type EarthlyConfig struct {
	Global EarthlyGlobalConfig `yaml:"global"`
}

// EarthlyGlobalConfig is the global configuration for Earthly.
type EarthlyGlobalConfig struct {
	BuildkitHost string `yaml:"buildkit_host"`
	TLSCA        string `yaml:"tlsca"`
	TLSCert      string `yaml:"tlscert"`
	TLSKey       string `yaml:"tlskey"`
}

// EarthlySatelliteOption is an option for configuring an EarthlySatellite.
type EarthlySatelliteOption func(*EarthlySatellite)

// WithCI sets the CI flag for the EarthlySatellite.
func WithCI(ci bool) EarthlySatelliteOption {
	return func(s *EarthlySatellite) {
		s.ci = ci
	}
}

// WithFs sets the filesystem for the EarthlySatellite.
func WithFs(fs fs.Filesystem) EarthlySatelliteOption {
	return func(s *EarthlySatellite) {
		s.fs = fs
	}
}

// WithSecretStore sets the secret store for the EarthlySatellite.
func WithSecretStore(secretStore secrets.SecretStore) EarthlySatelliteOption {
	return func(s *EarthlySatellite) {
		s.secretStore = secretStore
	}
}

// Configure configures the local system to use a remote Earthly Satellite.
func (s *EarthlySatellite) Configure() error {
	s.logger.Info("Loading Earthly satellite credentials")
	err := s.loadCredentials()
	if err != nil {
		return fmt.Errorf("failed to load credentials: %w", err)
	}

	s.logger.Info("Writing certificates")
	err = s.writeCerts()
	if err != nil {
		return fmt.Errorf("failed to write certificates: %w", err)
	}

	s.logger.Info("Generating Earthly config")
	cfg, err := s.generateConfig()
	if err != nil {
		return fmt.Errorf("failed to generate config: %w", err)
	}

	s.logger.Info("Writing Earthly config")
	err = s.fs.WriteFile(filepath.Join(s.path, "config.yml"), cfg, 0644)
	if err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	// Mask the host if we're running in CI.
	if s.ci {
		fmt.Printf("::add-mask::%s\n", s.creds.Host)
	}

	return nil
}

// loadCredentials loads the credentials for the EarthlySatellite.
func (s *EarthlySatellite) loadCredentials() error {
	if s.project.Blueprint.Global.Ci.Providers.Earthly == nil ||
		s.project.Blueprint.Global.Ci.Providers.Earthly.Satellite == nil ||
		s.project.Blueprint.Global.Ci.Providers.Earthly.Satellite.Credentials == nil {
		return fmt.Errorf("no satellite credentials found")
	}

	creds, err := providers.GetEarthlyProviderCreds(
		s.project.Blueprint.Global.Ci.Providers.Earthly.Satellite.Credentials,
		&s.secretStore,
		s.logger,
	)
	if err != nil {
		return fmt.Errorf("failed to get Earthly provider credentials: %w", err)
	}

	s.creds = creds
	return nil
}

// writeCerts writes the certificates to the filesystem.
func (s *EarthlySatellite) writeCerts() error {
	ca, err := base64.StdEncoding.DecodeString(s.creds.Ca)
	if err != nil {
		return fmt.Errorf("failed to decode ca.pem: %w", err)
	}

	s.logger.Debug("Writing ca.pem", "path", filepath.Join(s.path, "ca.pem"))
	err = s.fs.WriteFile(filepath.Join(s.path, "ca.pem"), ca, 0644)
	if err != nil {
		return fmt.Errorf("failed to write ca.pem: %w", err)
	}

	privateKey, err := base64.StdEncoding.DecodeString(s.creds.PrivateKey)
	if err != nil {
		return fmt.Errorf("failed to decode private key: %w", err)
	}

	s.logger.Debug("Writing key.pem", "path", filepath.Join(s.path, "key.pem"))
	err = s.fs.WriteFile(filepath.Join(s.path, "key.pem"), privateKey, 0600)
	if err != nil {
		return fmt.Errorf("failed to write private key: %w", err)
	}

	cert, err := base64.StdEncoding.DecodeString(s.creds.Certificate)
	if err != nil {
		return fmt.Errorf("failed to decode certificate: %w", err)
	}

	s.logger.Debug("Writing cert.pem", "path", filepath.Join(s.path, "cert.pem"))
	err = s.fs.WriteFile(filepath.Join(s.path, "cert.pem"), cert, 0644)
	if err != nil {
		return fmt.Errorf("failed to write certificate: %w", err)
	}

	return nil
}

// generateConfig generates the Earthly config file.
func (s *EarthlySatellite) generateConfig() ([]byte, error) {
	cfg := EarthlyConfig{
		Global: EarthlyGlobalConfig{
			BuildkitHost: s.creds.Host,
			TLSCA:        "ca.pem",
			TLSCert:      "cert.pem",
			TLSKey:       "key.pem",
		},
	}

	cfgBytes, err := yaml.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Earthly config: %w", err)
	}

	return cfgBytes, nil
}

// NewEarthlySatellite creates a new EarthlySatellite.
func NewEarthlySatellite(p *project.Project, configPath string, logger *slog.Logger, opts ...EarthlySatelliteOption) *EarthlySatellite {
	s := &EarthlySatellite{
		project:     p,
		path:        configPath,
		fs:          billy.NewBaseOsFS(),
		logger:      logger,
		secretStore: secrets.NewDefaultSecretStore(),
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}
