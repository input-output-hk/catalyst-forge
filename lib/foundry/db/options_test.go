package db

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestWithTxOptions(t *testing.T) {
	t.Parallel()

	t.Run("ok/with_txstore_support", func(t *testing.T) {
		t.Parallel()

		// Mock store that supports TxStore
		mockStore := &mockTxStore{
			withTxOptionsCalled: false,
		}

		opts := &sql.TxOptions{
			Isolation: sql.LevelSerializable,
			ReadOnly:  true,
		}

		err := WithTxOptions(context.Background(), mockStore, opts, func(ctx context.Context) error {
			return nil
		})

		require.NoError(t, err, "should succeed")
		assert.True(t, mockStore.withTxOptionsCalled, "WithTxOptions should be called")
	})

	t.Run("ok/fallback_to_withtx", func(t *testing.T) {
		t.Parallel()

		// Mock store that doesn't support TxStore
		mockStore := &mockBasicStore{
			withTxCalled: false,
		}

		opts := &sql.TxOptions{
			Isolation: sql.LevelSerializable,
			ReadOnly:  true,
		}

		err := WithTxOptions(context.Background(), mockStore, opts, func(ctx context.Context) error {
			return nil
		})

		require.NoError(t, err, "should succeed")
		assert.True(t, mockStore.withTxCalled, "WithTx should be called as fallback")
	})
}

// Mock implementations for testing

type mockTxStore struct {
	withTxOptionsCalled bool
}

func (m *mockTxStore) Read() *gorm.DB {
	return nil
}

func (m *mockTxStore) Write() *gorm.DB {
	return nil
}

func (m *mockTxStore) Ping(ctx context.Context) error {
	return nil
}

func (m *mockTxStore) WithTxOptions(ctx context.Context, opts *sql.TxOptions, fn func(ctx context.Context) error) error {
	m.withTxOptionsCalled = true
	return fn(ctx)
}

func (m *mockTxStore) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

type mockBasicStore struct {
	withTxCalled bool
}

func (m *mockBasicStore) Read() *gorm.DB {
	return nil
}

func (m *mockBasicStore) Write() *gorm.DB {
	return nil
}

func (m *mockBasicStore) Ping(ctx context.Context) error {
	return nil
}

func (m *mockBasicStore) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	m.withTxCalled = true
	return fn(ctx)
}