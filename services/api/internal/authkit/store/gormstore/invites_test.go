package gormstore

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestNewInviteStore(t *testing.T) {
	t.Parallel()
	
	db, _ := setupTestDB(t)
	store := NewInviteStore(db)
	assert.NotNil(t, store)
	assert.NotNil(t, store.db)
}

func TestInviteStore_Create(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	inviteID := uuid.New()
	createdBy := uuid.New()
	now := time.Now()
	future := now.Add(24 * time.Hour)
	
	tests := []struct {
		name    string
		invite  *domain.Invite
		setup   func(sqlmock.Sqlmock)
		wantErr bool
		errMsg  string
	}{
		{
			name: "ok/basic_invite",
			invite: &domain.Invite{
				ID:        inviteID,
				Email:     "user@example.com",
				Roles:     []string{"user"},
				TokenHash: []byte("token-hash"),
				ExpiresAt: future,
				Attempts:  0,
				CreatedBy: createdBy,
			},
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta(
					`INSERT INTO "auth_invites" ("id","email","roles","token_hash","expires_at","attempts","redeemed_at","created_by","created_at") VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
				)).WithArgs(
					inviteID,
					"user@example.com",
					`["user"]`,
					[]byte("token-hash"),
					sqlmock.AnyArg(), // ExpiresAt
					0,
					nil,
					createdBy,
					sqlmock.AnyArg(), // CreatedAt
				).WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			wantErr: false,
		},
		{
			name: "ok/multiple_roles",
			invite: &domain.Invite{
				ID:        inviteID,
				Email:     "admin@example.com",
				Roles:     []string{"user", "admin"},
				TokenHash: []byte("admin-token-hash"),
				ExpiresAt: future,
				Attempts:  0,
				CreatedBy: createdBy,
			},
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta(
					`INSERT INTO "auth_invites" ("id","email","roles","token_hash","expires_at","attempts","redeemed_at","created_by","created_at") VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
				)).WithArgs(
					inviteID,
					"admin@example.com",
					`["user","admin"]`,
					[]byte("admin-token-hash"),
					sqlmock.AnyArg(), // ExpiresAt
					0,
					nil,
					createdBy,
					sqlmock.AnyArg(), // CreatedAt
				).WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			wantErr: false,
		},
		{
			name: "error/duplicate_invite",
			invite: &domain.Invite{
				ID:        inviteID,
				Email:     "duplicate@example.com",
				Roles:     []string{"user"},
				TokenHash: []byte("duplicate-hash"),
				ExpiresAt: future,
				Attempts:  0,
				CreatedBy: createdBy,
			},
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta(
					`INSERT INTO "auth_invites" ("id","email","roles","token_hash","expires_at","attempts","redeemed_at","created_by","created_at") VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
				)).WithArgs(
					inviteID,
					"duplicate@example.com",
					`["user"]`,
					[]byte("duplicate-hash"),
					sqlmock.AnyArg(), // ExpiresAt
					0,
					nil,
					createdBy,
					sqlmock.AnyArg(), // CreatedAt
				).WillReturnError(errors.New("duplicate key value violates unique constraint"))
				mock.ExpectRollback()
			},
			wantErr: true,
			errMsg:  "duplicate key",
		},
		{
			name: "error/db_error",
			invite: &domain.Invite{
				ID:        inviteID,
				Email:     "error@example.com",
				Roles:     []string{"user"},
				TokenHash: []byte("error-hash"),
				ExpiresAt: future,
				Attempts:  0,
				CreatedBy: createdBy,
			},
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta(
					`INSERT INTO "auth_invites" ("id","email","roles","token_hash","expires_at","attempts","redeemed_at","created_by","created_at") VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
				)).WithArgs(
					inviteID,
					"error@example.com",
					`["user"]`,
					[]byte("error-hash"),
					sqlmock.AnyArg(), // ExpiresAt
					0,
					nil,
					createdBy,
					sqlmock.AnyArg(), // CreatedAt
				).WillReturnError(errors.New("database connection lost"))
				mock.ExpectRollback()
			},
			wantErr: true,
			errMsg:  "database connection lost",
		},
	}
	
	for _, tc := range tests {
		tc := tc // capture range variable for parallel tests
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			
			db, mock := setupTestDB(t)
			store := NewInviteStore(db)
			
			tc.setup(mock)
			
			err := store.Create(ctx, tc.invite)
			
			if tc.wantErr {
				require.Error(t, err)
				if tc.errMsg != "" {
					assert.Contains(t, err.Error(), tc.errMsg)
				}
			} else {
				require.NoError(t, err)
			}
			
			err = mock.ExpectationsWereMet()
			assert.NoError(t, err)
		})
	}
}

func TestInviteStore_Get(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	inviteID := uuid.New()
	createdBy := uuid.New()
	now := time.Now()
	future := now.Add(24 * time.Hour)
	
	tests := []struct {
		name    string
		id      uuid.UUID
		setup   func(sqlmock.Sqlmock)
		wantErr bool
		errMsg  string
		validate func(*testing.T, *domain.Invite)
	}{
		{
			name: "ok/found_invite",
			id:   inviteID,
			setup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"id", "email", "roles", "token_hash", "expires_at", "attempts", "redeemed_at", "created_by",
				}).AddRow(
					inviteID, "user@example.com", `["user"]`, []byte("token-hash"), future, 0, nil, createdBy,
				)
				
				mock.ExpectQuery(regexp.QuoteMeta(
					`SELECT * FROM "auth_invites" WHERE id = $1 ORDER BY "auth_invites"."id" LIMIT $2`,
				)).WithArgs(inviteID, 1).WillReturnRows(rows)
			},
			wantErr: false,
			validate: func(t *testing.T, invite *domain.Invite) {
				assert.Equal(t, inviteID, invite.ID)
				assert.Equal(t, "user@example.com", invite.Email)
				assert.Equal(t, []string{"user"}, invite.Roles)
				assert.Equal(t, []byte("token-hash"), invite.TokenHash)
				assert.Equal(t, 0, invite.Attempts)
				assert.Nil(t, invite.RedeemedAt)
				assert.Equal(t, createdBy, invite.CreatedBy)
			},
		},
		{
			name: "ok/redeemed_invite",
			id:   inviteID,
			setup: func(mock sqlmock.Sqlmock) {
				redeemedAt := now.Add(-1 * time.Hour)
				rows := sqlmock.NewRows([]string{
					"id", "email", "roles", "token_hash", "expires_at", "attempts", "redeemed_at", "created_by",
				}).AddRow(
					inviteID, "user@example.com", `["user","admin"]`, []byte("token-hash"), future, 3, redeemedAt, createdBy,
				)
				
				mock.ExpectQuery(regexp.QuoteMeta(
					`SELECT * FROM "auth_invites" WHERE id = $1 ORDER BY "auth_invites"."id" LIMIT $2`,
				)).WithArgs(inviteID, 1).WillReturnRows(rows)
			},
			wantErr: false,
			validate: func(t *testing.T, invite *domain.Invite) {
				assert.Equal(t, []string{"user", "admin"}, invite.Roles)
				assert.Equal(t, 3, invite.Attempts)
				assert.NotNil(t, invite.RedeemedAt)
			},
		},
		{
			name: "error/not_found",
			id:   uuid.New(),
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(
					`SELECT * FROM "auth_invites" WHERE id = $1 ORDER BY "auth_invites"."id" LIMIT $2`,
				)).WithArgs(sqlmock.AnyArg(), 1).WillReturnError(gorm.ErrRecordNotFound)
			},
			wantErr: true,
			errMsg:  "invite not found",
		},
		{
			name: "error/db_error",
			id:   inviteID,
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(
					`SELECT * FROM "auth_invites" WHERE id = $1 ORDER BY "auth_invites"."id" LIMIT $2`,
				)).WithArgs(inviteID, 1).WillReturnError(errors.New("connection timeout"))
			},
			wantErr: true,
			errMsg:  "connection timeout",
		},
	}
	
	for _, tc := range tests {
		tc := tc // capture range variable for parallel tests
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			
			db, mock := setupTestDB(t)
			store := NewInviteStore(db)
			
			tc.setup(mock)
			
			invite, err := store.Get(ctx, tc.id)
			
			if tc.wantErr {
				require.Error(t, err)
				if tc.errMsg != "" {
					assert.Contains(t, err.Error(), tc.errMsg)
				}
				assert.Nil(t, invite)
			} else {
				require.NoError(t, err)
				require.NotNil(t, invite)
				
				if tc.validate != nil {
					tc.validate(t, invite)
				}
			}
			
			err = mock.ExpectationsWereMet()
			assert.NoError(t, err)
		})
	}
}

func TestInviteStore_IncrementAttempts(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	inviteID := uuid.New()
	
	tests := []struct {
		name    string
		id      uuid.UUID
		setup   func(sqlmock.Sqlmock)
		wantErr bool
		errMsg  string
	}{
		{
			name: "ok/increment_attempts",
			id:   inviteID,
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta(
					`UPDATE "auth_invites" SET "attempts"=attempts + $1 WHERE id = $2 AND redeemed_at IS NULL`,
				)).WithArgs(
					1,
					inviteID,
				).WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit()
			},
			wantErr: false,
		},
		{
			name: "error/invite_not_found",
			id:   uuid.New(),
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta(
					`UPDATE "auth_invites" SET "attempts"=attempts + $1 WHERE id = $2 AND redeemed_at IS NULL`,
				)).WithArgs(
					1,
					sqlmock.AnyArg(),
				).WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectCommit()
			},
			wantErr: true,
			errMsg:  "invite not found or already redeemed",
		},
		{
			name: "error/db_error",
			id:   inviteID,
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta(
					`UPDATE "auth_invites" SET "attempts"=attempts + $1 WHERE id = $2 AND redeemed_at IS NULL`,
				)).WithArgs(
					1,
					inviteID,
				).WillReturnError(errors.New("deadlock detected"))
				mock.ExpectRollback()
			},
			wantErr: true,
			errMsg:  "deadlock detected",
		},
	}
	
	for _, tc := range tests {
		tc := tc // capture range variable for parallel tests
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			
			db, mock := setupTestDB(t)
			store := NewInviteStore(db)
			
			tc.setup(mock)
			
			err := store.IncrementAttempts(ctx, tc.id)
			
			if tc.wantErr {
				require.Error(t, err)
				if tc.errMsg != "" {
					assert.Contains(t, err.Error(), tc.errMsg)
				}
			} else {
				require.NoError(t, err)
			}
			
			err = mock.ExpectationsWereMet()
			assert.NoError(t, err)
		})
	}
}

func TestInviteStore_Redeem(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	inviteID := uuid.New()
	now := time.Now()
	
	tests := []struct {
		name    string
		id      uuid.UUID
		at      time.Time
		setup   func(sqlmock.Sqlmock)
		wantErr bool
		errMsg  string
	}{
		{
			name: "ok/redeem_invite",
			id:   inviteID,
			at:   now,
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta(
					`UPDATE "auth_invites" SET "redeemed_at"=$1 WHERE id = $2 AND redeemed_at IS NULL`,
				)).WithArgs(
					sqlmock.AnyArg(),
					inviteID,
				).WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit()
			},
			wantErr: false,
		},
		{
			name: "error/invite_not_found",
			id:   uuid.New(),
			at:   now,
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta(
					`UPDATE "auth_invites" SET "redeemed_at"=$1 WHERE id = $2 AND redeemed_at IS NULL`,
				)).WithArgs(
					sqlmock.AnyArg(),
					sqlmock.AnyArg(),
				).WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectCommit()
			},
			wantErr: true,
			errMsg:  "invite not found or already redeemed",
		},
		{
			name: "error/db_error",
			id:   inviteID,
			at:   now,
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta(
					`UPDATE "auth_invites" SET "redeemed_at"=$1 WHERE id = $2 AND redeemed_at IS NULL`,
				)).WithArgs(
					sqlmock.AnyArg(),
					inviteID,
				).WillReturnError(errors.New("disk full"))
				mock.ExpectRollback()
			},
			wantErr: true,
			errMsg:  "disk full",
		},
	}
	
	for _, tc := range tests {
		tc := tc // capture range variable for parallel tests
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			
			db, mock := setupTestDB(t)
			store := NewInviteStore(db)
			
			tc.setup(mock)
			
			err := store.Redeem(ctx, tc.id, tc.at)
			
			if tc.wantErr {
				require.Error(t, err)
				if tc.errMsg != "" {
					assert.Contains(t, err.Error(), tc.errMsg)
				}
			} else {
				require.NoError(t, err)
			}
			
			err = mock.ExpectationsWereMet()
			assert.NoError(t, err)
		})
	}
}

func TestInviteStore_toDomain(t *testing.T) {
	t.Parallel()
	
	inviteID := uuid.New()
	createdBy := uuid.New()
	now := time.Now()
	future := now.Add(24 * time.Hour)
	
	tests := []struct {
		name    string
		invite  *Invite
		want    *domain.Invite
		wantErr bool
		errMsg  string
	}{
		{
			name: "ok/basic_invite",
			invite: &Invite{
				ID:         inviteID,
				Email:      "user@example.com",
				RolesJSON:  `["user"]`,
				TokenHash:  []byte("token-hash"),
				ExpiresAt:  future,
				Attempts:   0,
				RedeemedAt: nil,
				CreatedBy:  createdBy,
			},
			want: &domain.Invite{
				ID:         inviteID,
				Email:      "user@example.com",
				Roles:      []string{"user"},
				TokenHash:  []byte("token-hash"),
				ExpiresAt:  future,
				Attempts:   0,
				RedeemedAt: nil,
				CreatedBy:  createdBy,
			},
			wantErr: false,
		},
		{
			name: "ok/empty_roles",
			invite: &Invite{
				ID:         inviteID,
				Email:      "user@example.com",
				RolesJSON:  "",
				TokenHash:  []byte("token-hash"),
				ExpiresAt:  future,
				Attempts:   0,
				RedeemedAt: nil,
				CreatedBy:  createdBy,
			},
			want: &domain.Invite{
				ID:         inviteID,
				Email:      "user@example.com",
				Roles:      nil,
				TokenHash:  []byte("token-hash"),
				ExpiresAt:  future,
				Attempts:   0,
				RedeemedAt: nil,
				CreatedBy:  createdBy,
			},
			wantErr: false,
		},
		{
			name: "ok/invalid_json_treated_as_empty",
			invite: &Invite{
				ID:         inviteID,
				Email:      "user@example.com",
				RolesJSON:  `{invalid json}`,
				TokenHash:  []byte("token-hash"),
				ExpiresAt:  future,
				Attempts:   0,
				RedeemedAt: nil,
				CreatedBy:  createdBy,
			},
			want: &domain.Invite{
				ID:         inviteID,
				Email:      "user@example.com",
				Roles:      []string{},
				TokenHash:  []byte("token-hash"),
				ExpiresAt:  future,
				Attempts:   0,
				RedeemedAt: nil,
				CreatedBy:  createdBy,
			},
			wantErr: false,
		},
	}
	
	for _, tc := range tests {
		tc := tc // capture range variable for parallel tests
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			
			db, _ := setupTestDB(t)
			store := NewInviteStore(db)
			
			got, err := store.toDomain(tc.invite)
			
			if tc.wantErr {
				require.Error(t, err)
				if tc.errMsg != "" {
					assert.Contains(t, err.Error(), tc.errMsg)
				}
				assert.Nil(t, got)
			} else {
				require.NoError(t, err)
				require.NotNil(t, got)
				assert.Equal(t, tc.want.ID, got.ID)
				assert.Equal(t, tc.want.Email, got.Email)
				assert.Equal(t, tc.want.Roles, got.Roles)
				assert.Equal(t, tc.want.TokenHash, got.TokenHash)
				assert.Equal(t, tc.want.Attempts, got.Attempts)
				assert.Equal(t, tc.want.RedeemedAt, got.RedeemedAt)
				assert.Equal(t, tc.want.CreatedBy, got.CreatedBy)
			}
		})
	}
}