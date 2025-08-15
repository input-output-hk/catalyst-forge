package gormstore

import (
	"context"
	"fmt"

	repodb "github.com/catalystgo/catalyst-forge/lib/foundry/db"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/authkit/store"
	"gorm.io/gorm"
)

// Stores aggregates all GORM-based store implementations.
type Stores struct {
	Users         store.UserStore
	Credentials   store.CredentialStore
	Invites       store.InviteStore
	RefreshTokens store.RefreshStore
	RecoveryCodes store.RecoveryCodeStore
	Audit         store.AuditStore
}

// NewStores creates all GORM-based stores using the provided database handle.
//
// This is typically used with the Write() handle from a db.Store:
//
//	dbStore, err := db.Open(ctx, cfg)
//	if err != nil {
//	    return err
//	}
//	stores := NewStores(dbStore.Write())
func NewStores(db *gorm.DB) *Stores {
	return &Stores{
		Users:         NewUserStore(db),
		Credentials:   NewCredentialStore(db),
		Invites:       NewInviteStore(db),
		RefreshTokens: NewRefreshStore(db),
		RecoveryCodes: NewRecoveryCodeStore(db),
		Audit:         NewAuditStore(db),
	}
}

// Setup initializes the database with authkit stores and runs migrations.
//
// Example:
//
//	cfg := db.Config{
//	    DSN:          "postgres://localhost/mydb",
//	    MaxOpenConns: 10,
//	}
//
//	dbStore, stores, err := Setup(ctx, cfg)
//	if err != nil {
//	    return err
//	}
//	defer db.Close(dbStore)
func Setup(ctx context.Context, cfg repodb.Config) (repodb.Store, *Stores, error) {
	// Open database connection
	dbStore, err := repodb.Open(ctx, cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Run migrations
	if err := repodb.RunMigrations(dbStore.Write(), AutoMigrate); err != nil {
		repodb.Close(dbStore)
		return nil, nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	// Create stores
	stores := NewStores(dbStore.Write())

	return dbStore, stores, nil
}

// WithTransaction executes a function within a database transaction.
//
// All store operations within fn will share the same transaction.
// The transaction is automatically committed if fn returns nil,
// or rolled back if fn returns an error.
//
// Example:
//
//	err := WithTransaction(ctx, dbStore, func(ctx context.Context) error {
//	    // All operations here share the same transaction
//	    user, err := stores.Users.Create(ctx, "user@example.com", []string{"user"})
//	    if err != nil {
//	        return err // Transaction will rollback
//	    }
//
//	    return stores.Audit.Record(ctx, domain.Event{
//	        Type:   domain.EventUserCreated,
//	        UserID: user.ID,
//	    })
//	})
func WithTransaction(ctx context.Context, dbStore repodb.Store, fn func(ctx context.Context) error) error {
	return dbStore.WithTx(ctx, fn)
}
