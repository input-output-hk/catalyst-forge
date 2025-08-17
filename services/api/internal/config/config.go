package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"
)

// Config represents the application configuration.
type Config struct {
	Server     ServerConfig
	Auth       AuthConfig
	Database   DatabaseConfig
	Logging    LoggingConfig
	Kubernetes KubernetesConfig
	Email      EmailConfig
	Security   SecurityConfig
	Certs      CertsConfig
}

// ServerConfig represents server-specific configuration.
type ServerConfig struct {
	HttpPort       int
	Timeout        time.Duration
	PublicBaseURL  string
	CookieSameSite string
}

// AuthConfig represents authentication-specific configuration.
type AuthConfig struct {
	// Token TTL configuration
	InviteTTL  time.Duration
	AccessTTL  time.Duration
	RefreshTTL time.Duration
	StepUpTTL  time.Duration

	// WebAuthn / cookies
	RPName              string
	RequireUV           bool
	ChallengeTTL        time.Duration
	RefreshCookieName   string
	RefreshCookieSecure bool

	// Rate limiting for auth endpoints (boolean toggle)
	RateEnabled bool

	// Rate limiting for invite verification
	InviteMaxAttempts int

	// Bootstrap / JWKS
	BootstrapToken string
	JWKSRoute      bool

	// Persistent signing keys (optional; if unset, ephemeral key is generated)
	SigningKeyPath string // Path to PEM-encoded ES256 private key
	SigningKeyPEM  string // Inline PEM-encoded ES256 private key
	SigningKeyKID  string // Key ID to use for signing/JWKS

	// CSRF secret (optional; if unset, random secret is generated on boot)
	// Accepts raw string, base64 (std or raw-url) encoded bytes, or hex.
	CSRFSecret string

	// RBAC defaults seeding
	RBACSeedDefaults bool

	// Development mode - NEVER enable in production
	DevMode bool

	// Admin policy
	AdminAAGUIDAllowlist []string

	// GitHub OIDC
	GitHub struct {
		Enabled      bool
		Issuer       string
		Audiences    []string
		JWKSCacheTTL time.Duration
		ExchangeTTL  time.Duration
	}
}

// EmailConfig represents outbound email configuration.
type EmailConfig struct {
	Enabled   bool
	Provider  string
	Sender    string
	SESRegion string
}

// SecurityConfig toggles security-related features.
type SecurityConfig struct {
	EnableNaivePerIPRateLimit bool
}

// DatabaseConfig represents database-specific configuration.
type DatabaseConfig struct {
	Host     string
	DbPort   int
	User     string
	Password string
	Name     string
	SSLMode  string
}

// LoggingConfig represents logging-specific configuration.
type LoggingConfig struct {
	Level  string
	Format string
}

// KubernetesConfig represents Kubernetes-specific configuration.
type KubernetesConfig struct {
	Namespace string
	Enabled   bool
}

// CertsConfig represents configuration for certificate issuance feature.
type CertsConfig struct {
	// ACM-PCA configuration
	PCAClientCAArn       string
	PCAServerCAArn       string
	PCAClientTemplateArn string
	PCAServerTemplateArn string
	PCASigningAlgoClient string
	PCASigningAlgoServer string
	PCATimeout           time.Duration

	// Policy
	ClientCertTTLDev   time.Duration
	ClientCertTTLCIMax time.Duration
	ServerCertTTL      time.Duration
	IssuanceRateHourly int
	SessionMaxActive   int
	RequirePermsAnd    bool

	// Optional CA register (S3 + DynamoDB)
	CARegion   string
	CADDBTable string
	CAS3Bucket string
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	// Validate required fields
	if c.Database.Password == "" {
		return errors.New("database password is required (use --password or DB_PASSWORD env var)")
	}
	// Validate bootstrap token if provided
	if c.Auth.BootstrapToken != "" && len(c.Auth.BootstrapToken) < 32 {
		return errors.New("BootstrapToken must be at least 32 characters long for security")
	}
	return nil
}

func isLocalhost(url string) bool {
	// crude check for localhost or 127.*
	return url == "http://localhost" || url == "https://localhost" ||
		len(url) >= 16 && url[:16] == "http://localhost" ||
		len(url) >= 17 && url[:17] == "https://localhost" ||
		len(url) >= 11 && url[:11] == "http://127.0" ||
		len(url) >= 12 && url[:12] == "https://127.0"
}

// GetDSN returns the database connection string.
func (c *Config) GetDSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Database.Host,
		c.Database.DbPort,
		c.Database.User,
		c.Database.Password,
		c.Database.Name,
		c.Database.SSLMode,
	)
}

// GetServerAddr returns the server address string.
func (c *Config) GetServerAddr() string {
	return fmt.Sprintf(":%d", c.Server.HttpPort)
}

// MaskSensitive returns a string indicating if a sensitive field is set.
func MaskSensitive(value string) string {
	if value == "" {
		return "<not set>"
	}
	if len(value) <= 8 {
		return "<set>"
	}
	// Show first 4 chars for debugging, rest masked
	return value[:4] + "****"
}

// GetSafeBootstrapInfo returns safe-to-log bootstrap token info.
func (c *Config) GetSafeBootstrapInfo() string {
	if c.Auth.BootstrapToken == "" {
		return "Bootstrap token not configured"
	}
	return fmt.Sprintf("Bootstrap token configured (length: %d)", len(c.Auth.BootstrapToken))
}

// GetLogger creates a slog.Logger based on the logging configuration.
func (c *Config) GetLogger() (*slog.Logger, error) {
	var level slog.Level
	switch c.Logging.Level {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		return nil, fmt.Errorf("unknown log level: %s", c.Logging.Level)
	}

	var handler slog.Handler
	switch c.Logging.Format {
	case "json":
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	case "text":
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	default:
		return nil, fmt.Errorf("unknown log format: %s", c.Logging.Format)
	}

	return slog.New(handler), nil
}
