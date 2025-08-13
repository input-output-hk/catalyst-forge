package dbtest

import (
	"context"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	postgresdriver "gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// OpenPostgres starts an ephemeral PostgreSQL container for integration testing.
//
// It uses Testcontainers to spin up a real PostgreSQL instance, configure it,
// and return a connected GORM database handle. The container is automatically
// terminated when the test completes.
func OpenPostgres(t *testing.T) *gorm.DB {
	t.Helper()

	ctx := context.Background()

	pgContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("failed to start postgres container: %v", err)
	}

	t.Cleanup(func() {
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Logf("failed to terminate postgres container: %v", err)
		}
	})

	dsn, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("failed to get connection string: %v", err)
	}

	db, err := gorm.Open(postgresdriver.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("failed to get underlying sql.DB: %v", err)
	}

	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(time.Hour)

	if err := sqlDB.Ping(); err != nil {
		t.Fatalf("failed to ping test database: %v", err)
	}

	return db
}

// OpenPostgresWithMigrations starts a PostgreSQL container and runs migrations.
//
// This is a convenience function that combines OpenPostgres with migration
// execution, useful for tests that require a specific database schema.
func OpenPostgresWithMigrations(t *testing.T, migrations ...func(*gorm.DB) error) *gorm.DB {
	t.Helper()

	db := OpenPostgres(t)

	for _, migrate := range migrations {
		if err := migrate(db); err != nil {
			t.Fatalf("failed to run migration: %v", err)
		}
	}

	return db
}
