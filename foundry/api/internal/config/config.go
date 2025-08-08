package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"
)

// Config represents the application configuration
type Config struct {
	Server     ServerConfig     `kong:"embed"`
	Auth       AuthConfig       `kong:"embed"`
	Database   DatabaseConfig   `kong:"embed"`
	Logging    LoggingConfig    `kong:"embed"`
	Kubernetes KubernetesConfig `kong:"embed,prefix='k8s-'"`
	StepCA     StepCAConfig     `kong:"embed"`
	Email      EmailConfig      `kong:"embed,prefix='email-'"`
	Security   SecurityConfig   `kong:"embed"`
}

// ServerConfig represents server-specific configuration
type ServerConfig struct {
	HttpPort      int           `kong:"help='HTTP port to listen on',default=8080,name='http-port',env='HTTP_PORT'"`
	Timeout       time.Duration `kong:"help='Server timeout',default=30s,env='SERVER_TIMEOUT'"`
	PublicBaseURL string        `kong:"help='Public base URL for generating links (e.g., https://api.example.com)',env='PUBLIC_BASE_URL'"`
}

// AuthConfig represents authentication-specific configuration
type AuthConfig struct {
	PrivateKey string        `kong:"help='Path to private key for JWT authentication',env='AUTH_PRIVATE_KEY'"`
	PublicKey  string        `kong:"help='Path to public key for JWT authentication',env='AUTH_PUBLIC_KEY'"`
	InviteTTL  time.Duration `kong:"help='Default invite TTL (e.g., 72h)',default=72h,env='INVITE_TTL'"`
	AccessTTL  time.Duration `kong:"help='Access token TTL (e.g., 30m)',default=30m,env='AUTH_ACCESS_TTL'"`
	RefreshTTL time.Duration `kong:"help='Default refresh token TTL (CLI/browser; used as base for rotation)',default=720h,env='AUTH_REFRESH_TTL'"`
	KETTTL     time.Duration `kong:"help='Key Enrollment Token TTL (e.g., 10m)',default=10m,env='KET_TTL'"`
}

// EmailConfig represents outbound email configuration
type EmailConfig struct {
	Enabled   bool   `kong:"help='Enable outbound emails',default=false,env='EMAIL_ENABLED'"`
	Provider  string `kong:"help='Email provider (ses, none)',default='none',env='EMAIL_PROVIDER'"`
	Sender    string `kong:"help='Sender email address',env='EMAIL_SENDER'"`
	SESRegion string `kong:"help='AWS SES region (e.g., us-east-1)',env='SES_REGION'"`
}

// SecurityConfig toggles security-related features
type SecurityConfig struct {
	EnableNaivePerIPRateLimit bool `kong:"help='Enable in-process per-IP rate limiting (not suitable behind proxies that hide client IP)',default=false,env='ENABLE_PER_IP_RATELIMIT'"`
}

// DatabaseConfig represents database-specific configuration
type DatabaseConfig struct {
	Host     string `kong:"help='Database host',default='localhost',env='DB_HOST'"`
	DbPort   int    `kong:"help='Database port',default=5432,name='db-port',env='DB_PORT'"`
	User     string `kong:"help='Database user',default='postgres',env='DB_USER'"`
	Password string `kong:"help='Database password',env='DB_PASSWORD'"`
	Name     string `kong:"help='Database name',default='releases',env='DB_NAME'"`
	SSLMode  string `kong:"help='Database SSL mode',default='disable',env='DB_SSLMODE'"`
}

// LoggingConfig represents logging-specific configuration
type LoggingConfig struct {
	Level  string `kong:"help='Log level (debug, info, warn, error)',default='info',env='LOG_LEVEL'"`
	Format string `kong:"help='Log format (json, text)',default='json',env='LOG_FORMAT'"`
}

// KubernetesConfig represents Kubernetes-specific configuration
type KubernetesConfig struct {
	Namespace string `kong:"help='Kubernetes namespace to use',default='default',env='K8S_NAMESPACE'"`
	Enabled   bool   `kong:"help='Enable Kubernetes integration',default=false,env='K8S_ENABLED'"`
}

// StepCAConfig represents step-ca certificate authority configuration
type StepCAConfig struct {
	BaseURL            string        `kong:"help='step-ca server URL',default='https://step-ca:9000',env='STEPCA_BASE_URL'"`
	InsecureSkipVerify bool          `kong:"help='Skip TLS certificate verification (only for testing!)',default=false,env='STEPCA_INSECURE_SKIP_VERIFY'"`
	ClientTimeout      time.Duration `kong:"help='Request timeout for step-ca client',default=30s,env='STEPCA_TIMEOUT',name='stepca-timeout'"`
	RootCA             string        `kong:"help='Path to step-ca root certificate for TLS verification',env='STEPCA_ROOT_CA'"`
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate required fields
	if c.Database.Password == "" {
		return errors.New("database password is required (use --password or DB_PASSWORD env var)")
	}
	return nil
}

// GetDSN returns the database connection string
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

// GetServerAddr returns the server address string
func (c *Config) GetServerAddr() string {
	return fmt.Sprintf(":%d", c.Server.HttpPort)
}

// GetLogger creates a slog.Logger based on the logging configuration
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
