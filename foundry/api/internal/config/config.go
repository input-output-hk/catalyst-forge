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
	Database   DatabaseConfig   `kong:"embed"`
	Logging    LoggingConfig    `kong:"embed"`
	Kubernetes KubernetesConfig `kong:"embed"`
}

// ServerConfig represents server-specific configuration
type ServerConfig struct {
	HttpPort int           `kong:"help='HTTP port to listen on',default=8080,name='http-port',env='HTTP_PORT'"`
	Timeout  time.Duration `kong:"help='Server timeout',default=30s,env='SERVER_TIMEOUT'"`
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
