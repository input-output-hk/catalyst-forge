package db

import (
	"context"

	"gorm.io/gorm"
)

// ctxKey is the type for context keys used by this package.
type ctxKey int

// txKey is the context key for storing transaction handles.
const txKey ctxKey = iota

// contextWithTx returns a new context with the transaction handle attached.
func contextWithTx(ctx context.Context, tx *gorm.DB) context.Context {
	return context.WithValue(ctx, txKey, tx)
}

// TxFromContext retrieves a transaction handle from the context.
//
// Returns nil if no transaction is present in the context.
// This is typically used by repositories to participate in
// ambient transactions started by Store.WithTx.
func TxFromContext(ctx context.Context) *gorm.DB {
	if tx, ok := ctx.Value(txKey).(*gorm.DB); ok {
		return tx
	}
	return nil
}
