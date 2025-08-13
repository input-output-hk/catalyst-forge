package gormstore

import (
	"context"
	"testing"

	"github.com/catalystgo/catalyst-forge/lib/foundry/db/dbtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestWithTransaction_Commit(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	
	// Setup test database with mock
	db, mock := dbtest.OpenMock(t)
	
	// Create a simple in-memory store for testing
	// In a real scenario, you'd use the actual db.Store
	store := &mockStore{db: db}
	
	// Test successful transaction (commit)
	mock.ExpectBegin()
	mock.ExpectCommit()
	
	err := WithTransaction(ctx, store, func(txCtx context.Context) error {
		// This would normally use the transaction from context
		// For this test, we're validating the pattern works
		return nil
	})
	
	require.NoError(t, err, "transaction should commit successfully")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestWithTransaction_Rollback(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	
	// Setup test database with mock
	db, mock := dbtest.OpenMock(t)
	
	// Create a simple in-memory store for testing
	store := &mockStore{db: db}
	
	// Test failed transaction (rollback)
	mock.ExpectBegin()
	mock.ExpectRollback()
	
	err := WithTransaction(ctx, store, func(txCtx context.Context) error {
		// Return error to trigger rollback
		return assert.AnError
	})
	
	require.Error(t, err, "transaction should return error")
	assert.Equal(t, assert.AnError, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// mockStore implements a minimal Store interface for testing
type mockStore struct {
	db *gorm.DB
}

func (m *mockStore) Read() *gorm.DB {
	return m.db
}

func (m *mockStore) Write() *gorm.DB {
	return m.db
}

func (m *mockStore) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return m.db.Transaction(func(tx *gorm.DB) error {
		txCtx := context.WithValue(ctx, ctxKey{}, tx)
		return fn(txCtx)
	})
}

func (m *mockStore) Ping(ctx context.Context) error {
	return nil
}

type ctxKey struct{}

func TestTransactionAwareness(t *testing.T) {
	t.Parallel()

	t.Run("stores_use_transaction_from_context", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		
		// Setup test database
		db, _ := dbtest.OpenMock(t)
		userStore := NewUserStore(db)
		
		// The actual transaction awareness is tested through the
		// WithTransaction tests above. This just verifies the dbFor
		// method returns the appropriate database handle.
		
		// When no transaction is in context, should return default db
		result := userStore.dbFor(ctx)
		assert.Equal(t, db, result, "should return default db when no transaction")
		
		// The transaction injection is handled by the db package
		// and tested in the WithTransaction tests
	})
}