//go:build integration

package testutil

import (
	"context"
	"fmt"
	"time"

	client "github.com/input-output-hk/catalyst-forge/lib/foundry/client"
)

// Suite encapsulates long-lived dependencies for the test run (Postgres container and DB snapshot).
type Suite struct {
	PG *PG
}

// Env is a per-test fresh environment: a new API server instance and admin token.
type Env struct {
	Suite      *Suite
	Server     *APIServer
	AdminJWT   string
	AdminEmail string
}

// SuiteStart starts Postgres once.
func SuiteStart(ctx context.Context) (*Suite, error) {
	pg, err := StartPostgres(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to start test suite postgres: %w", err)
	}
	return &Suite{PG: pg}, nil
}

// SnapshotMigrations starts a one-off API to run migrations, then snapshots the DB for later fast restores.
func (s *Suite) SnapshotMigrations(ctx context.Context) error {
	cfg, err := DefaultTestConfig()
	if err != nil {
		return fmt.Errorf("failed to create test config: %w", err)
	}
	srv, err := StartAPIServer(ctx, cfg, s.PG)
	if err != nil {
		return fmt.Errorf("failed to start API server for migrations: %w", err)
	}
	// Give server a moment to finish migrations
	time.Sleep(200 * time.Millisecond)
	srv.Stop()
	if err := s.PG.Snapshot(ctx); err != nil {
		return fmt.Errorf("failed to snapshot database after migrations: %w", err)
	}
	return nil
}

// PerTestEnv restores snapshot, starts API fresh, and bootstraps admin.
func (s *Suite) PerTestEnv(ctx context.Context, adminEmail string) (*Env, error) {
	if err := s.PG.Restore(ctx); err != nil {
		return nil, fmt.Errorf("failed to restore database snapshot: %w", err)
	}

	cfg, err := DefaultTestConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create test config: %w", err)
	}

	srv, err := StartAPIServer(ctx, cfg, s.PG)
	if err != nil {
		return nil, fmt.Errorf("failed to start API server: %w", err)
	}

	jwt, err := BootstrapAdmin(ctx, srv.BaseURL, cfg.BootstrapToken, adminEmail)
	if err != nil {
		srv.Stop()
		return nil, fmt.Errorf("failed to bootstrap admin user: %w", err)
	}

	return &Env{Suite: s, Server: srv, AdminJWT: jwt, AdminEmail: adminEmail}, nil
}

// AdminClient creates a new client authenticated with the admin JWT token.
func (e *Env) AdminClient() client.Client {
	return client.NewClient(e.Server.BaseURL, client.WithToken(e.AdminJWT))
}

// BaseURL returns the base URL of the test server.
func (e *Env) BaseURL() string {
	return e.Server.BaseURL
}

// NewTestEnv creates a fresh test environment using a global suite (for use in test package).
func NewTestEnv(ctx context.Context, suite *Suite) (*Env, error) {
	email := "admin-" + RandomHex(6) + "@foundry.dev"
	return suite.PerTestEnv(ctx, email)
}

// Close stops the API server.
func (e *Env) Close() {
	if e != nil && e.Server != nil {
		e.Server.Stop()
	}
}

// SuiteStop terminates the Postgres container.
func (s *Suite) SuiteStop(ctx context.Context) error {
	if err := s.PG.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop test suite: %w", err)
	}
	return nil
}
