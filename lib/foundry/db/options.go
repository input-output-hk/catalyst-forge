package db

import (
	"context"
	"database/sql"

	"gorm.io/gorm"
)

// TxStore extends Store with transaction options support.
//
// This interface is optional and can be implemented by stores that
// need to support custom transaction isolation levels or read-only transactions.
// Use type assertion to check if a Store supports this interface.
type TxStore interface {
	Store
	// WithTxOptions executes fn within a database transaction with custom options.
	WithTxOptions(ctx context.Context, opts *sql.TxOptions, fn func(ctx context.Context) error) error
}

// WithTxOptions executes fn within a database transaction with custom options.
//
// This is a convenience function that checks if the store supports TxStore
// and falls back to regular WithTx if not.
func WithTxOptions(ctx context.Context, s Store, opts *sql.TxOptions, fn func(ctx context.Context) error) error {
	if txStore, ok := s.(TxStore); ok {
		return txStore.WithTxOptions(ctx, opts, fn)
	}
	// Fall back to regular transaction if options not supported
	return s.WithTx(ctx, fn)
}

// WithTxOptions implements TxStore.WithTxOptions.
func (s *store) WithTxOptions(ctx context.Context, opts *sql.TxOptions, fn func(ctx context.Context) error) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txCtx := contextWithTx(ctx, tx)
		return fn(txCtx)
	}, opts)
}