package gormstore

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestGithubPolicyStore_Selects(t *testing.T) {
	t.Parallel()

	db, mock := setupTestDB(t)
	s := NewGithubPolicyStore(db)
	ctx := context.Background()

	id := uuid.New()
	_ = id // used below

	// GetByID
	rows := sqlmock.NewRows([]string{"id", "repository", "refs", "environments", "workflows", "roles", "enabled"}).
		AddRow(id, "org/repo", `[]`, `[]`, `[]`, `[]`, true)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "auth_github_policies" WHERE id = $1 ORDER BY "auth_github_policies"."id" LIMIT $2`)).
		WithArgs(id, 1).WillReturnRows(rows)
	_, _ = s.GetByID(ctx, id)

	// LookupByRepository
	rows2 := sqlmock.NewRows([]string{"id", "repository", "refs", "environments", "workflows", "roles", "enabled"}).
		AddRow(uuid.New(), "org/repo", `[]`, `[]`, `[]`, `[]`, true)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "auth_github_policies" WHERE repository = $1`)).
		WithArgs("org/repo").WillReturnRows(rows2)
	_, _ = s.LookupByRepository(ctx, "org/repo")

	require.NoError(t, mock.ExpectationsWereMet())
}
