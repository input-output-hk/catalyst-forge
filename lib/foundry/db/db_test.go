package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestTxFromContext(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		setupCtx func() context.Context
		wantNil  bool
	}{
		{
			name: "ok/with_transaction",
			setupCtx: func() context.Context {
				ctx := context.Background()
				mockDB := &gorm.DB{}
				return contextWithTx(ctx, mockDB)
			},
			wantNil: false,
		},
		{
			name: "ok/without_transaction",
			setupCtx: func() context.Context {
				return context.Background()
			},
			wantNil: true,
		},
	}

	for _, tc := range tests {
		tc := tc // capture range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := tc.setupCtx()
			tx := TxFromContext(ctx)

			if tc.wantNil {
				assert.Nil(t, tx, "expected nil transaction")
			} else {
				assert.NotNil(t, tx, "expected transaction in context")
			}
		})
	}
}

func TestRunMigrations(t *testing.T) {
	t.Parallel()

	t.Run("ok/sequential_execution", func(t *testing.T) {
		t.Parallel()

		var called []int

		m1 := func(db *gorm.DB) error {
			called = append(called, 1)
			return nil
		}

		m2 := func(db *gorm.DB) error {
			called = append(called, 2)
			return nil
		}

		m3 := func(db *gorm.DB) error {
			called = append(called, 3)
			return nil
		}

		err := RunMigrations(nil, m1, m2, m3)
		require.NoError(t, err, "migrations should succeed")
		assert.Equal(t, []int{1, 2, 3}, called, "migrations should run in order")
	})

	t.Run("error/stops_on_failure", func(t *testing.T) {
		t.Parallel()

		var called []int

		m1 := func(db *gorm.DB) error {
			called = append(called, 1)
			return nil
		}

		m2 := func(db *gorm.DB) error {
			called = append(called, 2)
			return assert.AnError
		}

		m3 := func(db *gorm.DB) error {
			called = append(called, 3)
			return nil
		}

		err := RunMigrations(nil, m1, m2, m3)
		require.Error(t, err, "should return error from failed migration")
		assert.Contains(t, err.Error(), "migration 1 failed", "error should indicate which migration failed")
		assert.Equal(t, []int{1, 2}, called, "should stop at failed migration")
	})

	t.Run("ok/no_migrations", func(t *testing.T) {
		t.Parallel()

		err := RunMigrations(nil)
		require.NoError(t, err, "should succeed with no migrations")
	})
}
