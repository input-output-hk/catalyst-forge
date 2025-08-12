//go:build integration

package testutil

import (
    "context"
    "fmt"
    "time"

    tc "github.com/testcontainers/testcontainers-go"
    "github.com/testcontainers/testcontainers-go/modules/postgres"
    "github.com/testcontainers/testcontainers-go/wait"
)

// PG wraps a running Postgres testcontainer and helpers.
type PG struct {
    Ctr *postgres.PostgresContainer
}

// StartPostgres starts a Postgres container for tests.
func StartPostgres(ctx context.Context) (*PG, error) {
    ctr, err := postgres.RunContainer(ctx,
        tc.WithImage("postgres:16-alpine"),
        tc.WithWaitStrategy(wait.ForLog("database system is ready to accept connections").WithStartupTimeout(90*time.Second)),
        postgres.WithDatabase("foundry"),
        postgres.WithUsername("foundry"),
        postgres.WithPassword("changeme"),
    )
    if err != nil {
        return nil, fmt.Errorf("failed to start postgres container: %w", err)
    }
    return &PG{Ctr: ctr}, nil
}

// Stop terminates the Postgres container.
func (p *PG) Stop(ctx context.Context) error {
    if p == nil || p.Ctr == nil {
        return nil
    }
    if err := p.Ctr.Terminate(ctx); err != nil {
        return fmt.Errorf("failed to terminate postgres container: %w", err)
    }
    return nil
}

// DSN returns discrete connection parameters for API env.
func (p *PG) DSN(ctx context.Context) (host string, port string, user string, pass string, db string, sslmode string, err error) {
    host, err = p.Ctr.Host(ctx)
    if err != nil {
        err = fmt.Errorf("failed to get postgres host: %w", err)
        return
    }
    mp, err2 := p.Ctr.MappedPort(ctx, "5432/tcp")
    if err2 != nil {
        err = fmt.Errorf("failed to get postgres port: %w", err2)
        return
    }
    port = mp.Port()
    user = "foundry"
    pass = "changeme"
    db = "foundry"
    sslmode = "disable"
    return
}

// Snapshot creates a snapshot of the DB state for fast restore.
func (p *PG) Snapshot(ctx context.Context) error {
    // Small delay to ensure no pending connections are writing when snapshotting
    time.Sleep(100 * time.Millisecond)
    if err := p.Ctr.Snapshot(ctx); err != nil {
        return fmt.Errorf("failed to snapshot postgres container: %w", err)
    }
    return nil
}

// Restore resets the DB back to the last snapshot.
func (p *PG) Restore(ctx context.Context) error {
    if err := p.Ctr.Restore(ctx); err != nil {
        return fmt.Errorf("failed to restore postgres container snapshot: %w", err)
    }
    return nil
}
