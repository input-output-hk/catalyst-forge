package jwt

import (
	"log/slog"
	"time"

	"github.com/input-output-hk/catalyst-forge/lib/tools/fs"
)

// ManagerOption is a function that configures an ES256Manager
type ManagerOption func(*ES256Manager)

// WithManagerAudiences sets the default audiences for the manager
func WithManagerAudiences(audiences []string) ManagerOption {
	return func(m *ES256Manager) {
		m.audiences = audiences
	}
}

// WithManagerFilesystem sets the filesystem for the manager
func WithManagerFilesystem(fs fs.Filesystem) ManagerOption {
	return func(m *ES256Manager) {
		m.fs = fs
	}
}

// WithManagerIssuer sets the issuer for the manager
func WithManagerIssuer(issuer string) ManagerOption {
	return func(m *ES256Manager) {
		m.issuer = issuer
	}
}

// WithManagerLogger sets the logger for the manager
func WithManagerLogger(logger *slog.Logger) ManagerOption {
	return func(m *ES256Manager) {
		m.logger = logger
	}
}

// WithMaxAuthTokenTTL sets the maximum allowed TTL for authentication tokens
func WithMaxAuthTokenTTL(ttl time.Duration) ManagerOption {
	return func(m *ES256Manager) {
		m.maxAuthTokenTTL = ttl
	}
}

// WithMaxCertificateTTL sets the maximum allowed TTL for certificate signing tokens
func WithMaxCertificateTTL(ttl time.Duration) ManagerOption {
	return func(m *ES256Manager) {
		m.maxCertificateTTL = ttl
	}
}
