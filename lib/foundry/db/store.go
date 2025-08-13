package db

import (
	"context"
	"database/sql"
	"fmt"

	"gorm.io/gorm"
)

// Store provides database read/write handles and transaction support.
type Store interface {
	// Read returns a database handle for read operations.
	// Currently returns the same handle as Write(), but this design
	// allows for future read replica support without breaking changes.
	Read() *gorm.DB

	// Write returns a database handle for write operations.
	Write() *gorm.DB

	// WithTx executes fn within a database transaction.
	// TODO: Add WithTxOptions variant to support custom isolation levels
	// and read-only transactions when needed.
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error

	// Ping checks database connectivity with the provided context deadline.
	Ping(ctx context.Context) error
}

// store implements the Store interface.
type store struct {
	db *gorm.DB
}

// Read returns a database handle for read operations.
func (s *store) Read() *gorm.DB {
	return s.db
}

// Write returns a database handle for write operations.
func (s *store) Write() *gorm.DB {
	return s.db
}

// WithTx executes fn within a database transaction.
//
// The transaction is automatically committed if fn returns nil,
// or rolled back if fn returns an error. The transactional database
// handle is injected into the context and can be retrieved using
// TxFromContext within fn.
func (s *store) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txCtx := contextWithTx(ctx, tx)
		return fn(txCtx)
	}, &sql.TxOptions{})
}

// Ping verifies database connectivity using the given context.
func (s *store) Ping(ctx context.Context) error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}
	return sqlDB.PingContext(ctx)
}

// Close closes the underlying database connection.
func (s *store) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}
	return sqlDB.Close()
}
