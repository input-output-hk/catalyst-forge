package gormstore

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestBootstrapTokenStore_MarkUsed_And_IsTokenUsed(t *testing.T) {
	t.Parallel()

	db, mock := setupTestDB(t)
	s := NewBootstrapTokenStore(db)
	ctx := context.Background()

	hash := "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"

	// 1) IsTokenUsed -> not found
	// Return empty rows to simulate not found (gorm translates to ErrRecordNotFound internally only when using First)
	rowsNone := sqlmock.NewRows([]string{"token_hash_hex", "used_by_email", "used_at"})
	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "auth_bootstrap_tokens" WHERE token_hash_hex = $1 ORDER BY "auth_bootstrap_tokens"."token_hash_hex" LIMIT $2`,
	)).WithArgs(hash, 1).WillReturnRows(rowsNone)

	used, err := s.IsTokenUsed(ctx, hash)
	require.NoError(t, err)
	require.False(t, used)

	// 2) MarkUsed -> insert row
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(
		`INSERT INTO "auth_bootstrap_tokens" ("token_hash_hex","used_by_email","used_at") VALUES ($1,$2,$3)`,
	)).WithArgs(hash, "admin@example.com", sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	require.NoError(t, s.MarkUsed(ctx, hash, "admin@example.com"))

	// 3) IsTokenUsed -> found
	rows := sqlmock.NewRows([]string{"token_hash_hex", "used_by_email", "used_at"}).AddRow(
		hash, "admin@example.com", time.Now(),
	)
	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "auth_bootstrap_tokens" WHERE token_hash_hex = $1 ORDER BY "auth_bootstrap_tokens"."token_hash_hex" LIMIT $2`,
	)).WithArgs(hash, 1).WillReturnRows(rows)

	used, err = s.IsTokenUsed(ctx, hash)
	require.NoError(t, err)
	require.True(t, used)

	// 4) MarkUsed again -> duplicate error
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(
		`INSERT INTO "auth_bootstrap_tokens" ("token_hash_hex","used_by_email","used_at") VALUES ($1,$2,$3)`,
	)).WithArgs(hash, "admin@example.com", sqlmock.AnyArg()).WillReturnError(assertAnError())
	mock.ExpectRollback()

	err = s.MarkUsed(ctx, hash, "admin@example.com")
	require.Error(t, err)

	require.NoError(t, mock.ExpectationsWereMet())
}

// gormErrRecordNotFound returns gorm.ErrRecordNotFound without importing gorm here
func gormErrRecordNotFound() error { return errRecordNotFound }

// assertAnError returns a generic error without adding extra imports in this file
func assertAnError() error { return anyError }
