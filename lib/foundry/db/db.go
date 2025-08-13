package db

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Config defines database connection and behavior settings.
type Config struct {
	// DSN is the database connection string.
	// For PostgreSQL, include statement_timeout in options:
	// "postgres://user:pass@host/db?options=-c statement_timeout=30000"
	DSN             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
	// StatementTimeout is deprecated. Configure timeout in DSN instead.
	// This field is kept for backward compatibility but not used.
	StatementTimeout time.Duration
	LogLevel         logger.LogLevel
	// SlowThreshold logs queries slower than this duration.
	// Set to 0 to disable slow query logging.
	SlowThreshold time.Duration
}

// Open initializes a Store with the provided configuration.
//
// It establishes a connection to the database, configures connection pooling,
// and verifies connectivity with a ping. Statement timeouts should be configured
// in the DSN (e.g., "postgres://localhost/mydb?options=-c statement_timeout=30000")
// to ensure they apply to all connections in the pool.
// The returned Store must be closed when no longer needed.
//
// Example:
//
//	cfg := Config{
//	    DSN:              "postgres://localhost/mydb?options=-c statement_timeout=30000",
//	    MaxOpenConns:     10,
//	    SlowThreshold:    200 * time.Millisecond,
//	}
//	store, err := Open(ctx, cfg)
//	if err != nil {
//	    return err
//	}
//	defer Close(store)
func Open(ctx context.Context, cfg Config) (Store, error) {
	// Configure GORM logger with slow threshold if specified
	gormLogger := logger.Default.LogMode(cfg.LogLevel)
	if cfg.SlowThreshold > 0 {
		gormLogger = logger.New(
			log.New(os.Stdout, "\r\n", log.LstdFlags),
			logger.Config{
				SlowThreshold:             cfg.SlowThreshold,
				LogLevel:                  cfg.LogLevel,
				IgnoreRecordNotFoundError: true,
				Colorful:                  false,
			},
		)
	}

	gormCfg := &gorm.Config{
		Logger: gormLogger,
	}

	db, err := gorm.Open(postgres.Open(cfg.DSN), gormCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	if cfg.MaxOpenConns > 0 {
		sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	}
	if cfg.MaxIdleConns > 0 {
		sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	}
	if cfg.ConnMaxLifetime > 0 {
		sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	}
	if cfg.ConnMaxIdleTime > 0 {
		sqlDB.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)
	}

	// Note: Statement timeout should be configured in the DSN for PostgreSQL
	// to ensure it applies to all connections in the pool. For example:
	// postgres://user:pass@host/db?options=-c statement_timeout=30000
	//
	// Alternatively, use AfterConnect callback in GORM v2.0+ to set it per connection:
	// gormCfg.ConnPool = ... (requires custom connection pool wrapper)

	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &store{
		db: db,
	}, nil
}

// RetryConfig controls retry/backoff behavior for OpenWithRetry and WaitUntilReady.
type RetryConfig struct {
	InitialBackoff time.Duration
	MaxBackoff     time.Duration
	Multiplier     float64
	MaxElapsed     time.Duration // 0 = unlimited (until ctx canceled)
}

func (rc RetryConfig) normalize() RetryConfig {
	out := rc
	if out.InitialBackoff <= 0 {
		out.InitialBackoff = 500 * time.Millisecond
	}
	if out.MaxBackoff <= 0 {
		out.MaxBackoff = 10 * time.Second
	}
	if out.Multiplier <= 1.0 {
		out.Multiplier = 1.5
	}
	return out
}

// OpenWithRetry attempts to open the DB and ping with exponential backoff until success or context cancellation.
func OpenWithRetry(ctx context.Context, cfg Config, rc RetryConfig) (Store, error) {
	rc = rc.normalize()
	start := time.Now()
	backoff := rc.InitialBackoff

	for {
		s, err := Open(ctx, cfg)
		if err == nil {
			return s, nil
		}

		if rc.MaxElapsed > 0 && time.Since(start) >= rc.MaxElapsed {
			return nil, err
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(backoff):
			if backoff < rc.MaxBackoff {
				next := time.Duration(float64(backoff) * rc.Multiplier)
				if next > rc.MaxBackoff {
					next = rc.MaxBackoff
				}
				backoff = next
			}
		}
	}
}

// WaitUntilReady pings an already-open Store until it responds or context is canceled.
func WaitUntilReady(ctx context.Context, s Store, rc RetryConfig) error {
	rc = rc.normalize()
	start := time.Now()
	backoff := rc.InitialBackoff

	for {
		pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		err := s.Ping(pingCtx)
		cancel()
		if err == nil {
			return nil
		}

		if rc.MaxElapsed > 0 && time.Since(start) >= rc.MaxElapsed {
			return err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
			if backoff < rc.MaxBackoff {
				next := time.Duration(float64(backoff) * rc.Multiplier)
				if next > rc.MaxBackoff {
					next = rc.MaxBackoff
				}
				backoff = next
			}
		}
	}
}

// Close closes the database connection associated with the Store.
//
// It is safe to call Close on a nil Store. If the Store implements
// a Close method, it will be called; otherwise, the underlying SQL
// database connection will be closed.
func Close(s Store) error {
	if s == nil {
		return nil
	}

	type closer interface {
		Close() error
	}

	if c, ok := s.(closer); ok {
		return c.Close()
	}

	sqlDB, err := s.Write().DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
