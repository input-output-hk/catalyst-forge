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
	Certs      CertsConfig      `kong:"embed,prefix='certs-'"`
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

// CertsConfig represents configuration for certificate issuance feature
type CertsConfig struct {
	// ACM-PCA configuration
	PCAClientCAArn       string        `kong:"help='ACM-PCA ARN for client certificates',env='PCA_CLIENT_CA_ARN'"`
	PCAServerCAArn       string        `kong:"help='ACM-PCA ARN for server certificates',env='PCA_SERVER_CA_ARN'"`
	PCAClientTemplateArn string        `kong:"help='ACM-PCA template ARN for client certs (APIPassthrough)',env='PCA_CLIENT_TEMPLATE_ARN'"`
	PCAServerTemplateArn string        `kong:"help='ACM-PCA template ARN for server certs (APIPassthrough)',env='PCA_SERVER_TEMPLATE_ARN'"`
	PCASigningAlgoClient string        `kong:"help='ACM-PCA SigningAlgorithm for client certs (e.g., SHA256WITHECDSA)',default='SHA256WITHECDSA',env='PCA_SIGNING_ALGO_CLIENT'"`
	PCASigningAlgoServer string        `kong:"help='ACM-PCA SigningAlgorithm for server certs (e.g., SHA256WITHECDSA)',default='SHA256WITHECDSA',env='PCA_SIGNING_ALGO_SERVER'"`
	PCATimeout           time.Duration `kong:"help='Timeout for ACM-PCA calls',default=10s,env='PCA_TIMEOUT'"`

	// Policy
	ClientCertTTLDev   time.Duration `kong:"help='Default TTL for developer client certs',default=90m,env='CLIENT_CERT_TTL_DEV'"`
	ClientCertTTLCIMax time.Duration `kong:"help='Maximum TTL for CI client certs',default=120m,env='CLIENT_CERT_TTL_CI_MAX'"`
	ServerCertTTL      time.Duration `kong:"help='TTL for server certificates',default=336h,env='SERVER_CERT_TTL'"`
	IssuanceRateHourly int           `kong:"help='Max certificate issuances per hour per subject/repo',default=6,env='ISSUANCE_RATE_HOURLY'"`
	SessionMaxActive   int           `kong:"help='Max concurrent build sessions per user',default=10,env='SESSION_MAX_ACTIVE'"`
	RequirePermsAnd    bool          `kong:"help='RequireAll authorization semantics globally',default=true,env='REQUIRE_PERMS_AND'"`
	// Feature flags
	ExtAuthzEnabled bool `kong:"help='Enable optional external authorization endpoint for BuildKit gateway',default=false,env='EXT_AUTHZ_ENABLED'"`

	// GitHub OIDC
	GhOIDCIssuer    string        `kong:"help='GitHub OIDC issuer',default='https://token.actions.githubusercontent.com',env='GITHUB_OIDC_ISS'"`
	GhOIDCAudience  string        `kong:"help='Expected audience for GitHub OIDC',default='forge',env='GITHUB_OIDC_AUD'"`
	GhAllowedOrgs   string        `kong:"help='Comma-separated allowed GitHub orgs',env='GITHUB_ALLOWED_ORGS'"`
	GhAllowedRepos  string        `kong:"help='Comma-separated allowed <org>/<repo> entries',env='GITHUB_ALLOWED_REPOS'"`
	GhProtectedRefs string        `kong:"help='Comma-separated protected refs (e.g., refs/heads/main,refs/tags/*)',env='GITHUB_PROTECTED_REFS'"`
	GhJWKSCacheTTL  time.Duration `kong:"help='JWKS cache TTL for GitHub OIDC',default=10m,env='GITHUB_JWKS_CACHE_TTL'"`

	// Job token minted by the API for CI after OIDC verification (no refresh)
	JobTokenDefaultTTL time.Duration `kong:"help='Default TTL for minted CI job tokens (clamped by OIDC token expiry)',default=60m,env='JOB_TOKEN_TTL'"`

	// Optional CA register (S3 + DynamoDB)
	CARegion   string `kong:"help='AWS region for CA register',env='CAREGION'"`
	CADDBTable string `kong:"help='DynamoDB table for CA register pointers',env='CA_DDB_TABLE'"`
	CAS3Bucket string `kong:"help='S3 bucket for CA register artifacts',env='CA_S3_BUCKET'"`
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
