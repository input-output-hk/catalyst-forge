package gormstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupTestDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	
	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn: sqlDB,
	}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	
	return gormDB, mock
}

func TestNewUserStore(t *testing.T) {
	t.Parallel()
	
	db, _ := setupTestDB(t)
	store := NewUserStore(db)
	assert.NotNil(t, store)
	assert.NotNil(t, store.db)
}

func TestUserStore_Create(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	
	tests := []struct {
		name    string
		email   string
		roles   []string
		setup   func(sqlmock.Sqlmock)
		wantErr bool
		errMsg  string
		validate func(*testing.T, *domain.User)
	}{
		{
			name:  "ok/basic_user",
			email: "user@example.com",
			roles: []string{"user"},
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta(
					`INSERT INTO "auth_users" ("id","email","roles","session_version","created_at","updated_at","deleted_at") VALUES ($1,$2,$3,$4,$5,$6,$7)`,
				)).WithArgs(
					sqlmock.AnyArg(), // ID (UUID)
					"user@example.com",
					`["user"]`,
					int64(1),
					sqlmock.AnyArg(), // CreatedAt
					sqlmock.AnyArg(), // UpdatedAt
					nil,              // DeletedAt
				).WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			wantErr: false,
			validate: func(t *testing.T, user *domain.User) {
				assert.NotEqual(t, uuid.Nil, user.ID)
				assert.Equal(t, "user@example.com", user.Email)
				assert.Equal(t, []string{"user"}, user.Roles)
				assert.Equal(t, int64(1), user.SessionVersion)
			},
		},
		{
			name:  "ok/admin_user",
			email: "admin@example.com",
			roles: []string{"user", "admin"},
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta(
					`INSERT INTO "auth_users" ("id","email","roles","session_version","created_at","updated_at","deleted_at") VALUES ($1,$2,$3,$4,$5,$6,$7)`,
				)).WithArgs(
					sqlmock.AnyArg(),
					"admin@example.com",
					`["user","admin"]`,
					int64(1),
					sqlmock.AnyArg(),
					sqlmock.AnyArg(),
					nil,
				).WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			wantErr: false,
			validate: func(t *testing.T, user *domain.User) {
				assert.Equal(t, "admin@example.com", user.Email)
				assert.Equal(t, []string{"user", "admin"}, user.Roles)
			},
		},
		{
			name:  "ok/no_roles",
			email: "noroles@example.com",
			roles: []string{},
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta(
					`INSERT INTO "auth_users" ("id","email","roles","session_version","created_at","updated_at","deleted_at") VALUES ($1,$2,$3,$4,$5,$6,$7)`,
				)).WithArgs(
					sqlmock.AnyArg(),
					"noroles@example.com",
					`[]`,
					int64(1),
					sqlmock.AnyArg(),
					sqlmock.AnyArg(),
					nil,
				).WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			wantErr: false,
			validate: func(t *testing.T, user *domain.User) {
				assert.Equal(t, []string{}, user.Roles)
			},
		},
		{
			name:  "error/duplicate_email",
			email: "duplicate@example.com",
			roles: []string{"user"},
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta(
					`INSERT INTO "auth_users" ("id","email","roles","session_version","created_at","updated_at","deleted_at") VALUES ($1,$2,$3,$4,$5,$6,$7)`,
				)).WithArgs(
					sqlmock.AnyArg(),
					"duplicate@example.com",
					`["user"]`,
					int64(1),
					sqlmock.AnyArg(),
					sqlmock.AnyArg(),
					nil,
				).WillReturnError(errors.New("duplicate key value violates unique constraint"))
				mock.ExpectRollback()
			},
			wantErr: true,
			errMsg:  "duplicate key",
		},
		{
			name:  "error/db_error",
			email: "error@example.com",
			roles: []string{"user"},
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta(
					`INSERT INTO "auth_users" ("id","email","roles","session_version","created_at","updated_at","deleted_at") VALUES ($1,$2,$3,$4,$5,$6,$7)`,
				)).WithArgs(
					sqlmock.AnyArg(),
					"error@example.com",
					`["user"]`,
					int64(1),
					sqlmock.AnyArg(),
					sqlmock.AnyArg(),
					nil,
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
			store := NewUserStore(db)
			
			tc.setup(mock)
			
			user, err := store.Create(ctx, tc.email, tc.roles)
			
			if tc.wantErr {
				require.Error(t, err)
				if tc.errMsg != "" {
					assert.Contains(t, err.Error(), tc.errMsg)
				}
				assert.Nil(t, user)
			} else {
				require.NoError(t, err)
				require.NotNil(t, user)
				
				if tc.validate != nil {
					tc.validate(t, user)
				}
			}
			
			err = mock.ExpectationsWereMet()
			assert.NoError(t, err)
		})
	}
}

func TestUserStore_GetByEmail(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	userID := uuid.New()
	now := time.Now()
	
	tests := []struct {
		name    string
		email   string
		setup   func(sqlmock.Sqlmock)
		wantErr bool
		errMsg  string
		validate func(*testing.T, *domain.User)
	}{
		{
			name:  "ok/found_user",
			email: "user@example.com",
			setup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"id", "email", "roles", "session_version", "created_at", "updated_at", "deleted_at",
				}).AddRow(
					userID, "user@example.com", `["user"]`, int64(1), now, now, nil,
				)
				
				mock.ExpectQuery(regexp.QuoteMeta(
					`SELECT * FROM "auth_users" WHERE email = $1 AND "auth_users"."deleted_at" IS NULL ORDER BY "auth_users"."id" LIMIT $2`,
				)).WithArgs("user@example.com", 1).WillReturnRows(rows)
			},
			wantErr: false,
			validate: func(t *testing.T, user *domain.User) {
				assert.Equal(t, userID, user.ID)
				assert.Equal(t, "user@example.com", user.Email)
				assert.Equal(t, []string{"user"}, user.Roles)
				assert.Equal(t, int64(1), user.SessionVersion)
			},
		},
		{
			name:  "ok/admin_user",
			email: "admin@example.com",
			setup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"id", "email", "roles", "session_version", "created_at", "updated_at", "deleted_at",
				}).AddRow(
					userID, "admin@example.com", `["user","admin"]`, int64(5), now, now, nil,
				)
				
				mock.ExpectQuery(regexp.QuoteMeta(
					`SELECT * FROM "auth_users" WHERE email = $1 AND "auth_users"."deleted_at" IS NULL ORDER BY "auth_users"."id" LIMIT $2`,
				)).WithArgs("admin@example.com", 1).WillReturnRows(rows)
			},
			wantErr: false,
			validate: func(t *testing.T, user *domain.User) {
				assert.Equal(t, []string{"user", "admin"}, user.Roles)
				assert.Equal(t, int64(5), user.SessionVersion)
			},
		},
		{
			name:  "error/not_found",
			email: "notfound@example.com",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(
					`SELECT * FROM "auth_users" WHERE email = $1 AND "auth_users"."deleted_at" IS NULL ORDER BY "auth_users"."id" LIMIT $2`,
				)).WithArgs("notfound@example.com", 1).WillReturnError(gorm.ErrRecordNotFound)
			},
			wantErr: true,
			errMsg:  "user not found",
		},
		{
			name:  "error/db_error",
			email: "error@example.com",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(
					`SELECT * FROM "auth_users" WHERE email = $1 AND "auth_users"."deleted_at" IS NULL ORDER BY "auth_users"."id" LIMIT $2`,
				)).WithArgs("error@example.com", 1).WillReturnError(errors.New("connection timeout"))
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
			store := NewUserStore(db)
			
			tc.setup(mock)
			
			user, err := store.GetByEmail(ctx, tc.email)
			
			if tc.wantErr {
				require.Error(t, err)
				if tc.errMsg != "" {
					assert.Contains(t, err.Error(), tc.errMsg)
				}
				assert.Nil(t, user)
			} else {
				require.NoError(t, err)
				require.NotNil(t, user)
				
				if tc.validate != nil {
					tc.validate(t, user)
				}
			}
			
			err = mock.ExpectationsWereMet()
			assert.NoError(t, err)
		})
	}
}

func TestUserStore_GetByID(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	userID := uuid.New()
	now := time.Now()
	
	tests := []struct {
		name    string
		id      uuid.UUID
		setup   func(sqlmock.Sqlmock)
		wantErr bool
		errMsg  string
		validate func(*testing.T, *domain.User)
	}{
		{
			name: "ok/found_user",
			id:   userID,
			setup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"id", "email", "roles", "session_version", "created_at", "updated_at", "deleted_at",
				}).AddRow(
					userID, "user@example.com", `["user"]`, int64(2), now, now, nil,
				)
				
				mock.ExpectQuery(regexp.QuoteMeta(
					`SELECT * FROM "auth_users" WHERE id = $1 AND "auth_users"."deleted_at" IS NULL ORDER BY "auth_users"."id" LIMIT $2`,
				)).WithArgs(userID, 1).WillReturnRows(rows)
			},
			wantErr: false,
			validate: func(t *testing.T, user *domain.User) {
				assert.Equal(t, userID, user.ID)
				assert.Equal(t, "user@example.com", user.Email)
				assert.Equal(t, int64(2), user.SessionVersion)
			},
		},
		{
			name: "error/not_found",
			id:   uuid.New(),
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(
					`SELECT * FROM "auth_users" WHERE id = $1 AND "auth_users"."deleted_at" IS NULL ORDER BY "auth_users"."id" LIMIT $2`,
				)).WithArgs(sqlmock.AnyArg(), 1).WillReturnError(gorm.ErrRecordNotFound)
			},
			wantErr: true,
			errMsg:  "user not found",
		},
		{
			name: "error/db_error",
			id:   userID,
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(
					`SELECT * FROM "auth_users" WHERE id = $1 AND "auth_users"."deleted_at" IS NULL ORDER BY "auth_users"."id" LIMIT $2`,
				)).WithArgs(userID, 1).WillReturnError(errors.New("disk full"))
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
			store := NewUserStore(db)
			
			tc.setup(mock)
			
			user, err := store.GetByID(ctx, tc.id)
			
			if tc.wantErr {
				require.Error(t, err)
				if tc.errMsg != "" {
					assert.Contains(t, err.Error(), tc.errMsg)
				}
				assert.Nil(t, user)
			} else {
				require.NoError(t, err)
				require.NotNil(t, user)
				
				if tc.validate != nil {
					tc.validate(t, user)
				}
			}
			
			err = mock.ExpectationsWereMet()
			assert.NoError(t, err)
		})
	}
}

func TestUserStore_UpdateRoles(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	userID := uuid.New()
	
	tests := []struct {
		name    string
		id      uuid.UUID
		roles   []string
		setup   func(sqlmock.Sqlmock)
		wantErr bool
		errMsg  string
	}{
		{
			name:  "ok/update_roles",
			id:    userID,
			roles: []string{"user", "admin"},
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta(
					`UPDATE "auth_users" SET "roles"=$1,"updated_at"=$2 WHERE id = $3 AND "auth_users"."deleted_at" IS NULL`,
				)).WithArgs(
					`["user","admin"]`,
					sqlmock.AnyArg(),
					userID,
				).WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit()
			},
			wantErr: false,
		},
		{
			name:  "ok/empty_roles",
			id:    userID,
			roles: []string{},
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta(
					`UPDATE "auth_users" SET "roles"=$1,"updated_at"=$2 WHERE id = $3 AND "auth_users"."deleted_at" IS NULL`,
				)).WithArgs(
					`[]`,
					sqlmock.AnyArg(),
					userID,
				).WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit()
			},
			wantErr: false,
		},
		{
			name:  "error/user_not_found",
			id:    uuid.New(),
			roles: []string{"user"},
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta(
					`UPDATE "auth_users" SET "roles"=$1,"updated_at"=$2 WHERE id = $3 AND "auth_users"."deleted_at" IS NULL`,
				)).WithArgs(
					`["user"]`,
					sqlmock.AnyArg(),
					sqlmock.AnyArg(),
				).WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectCommit()
			},
			wantErr: true,
			errMsg:  "user not found",
		},
		{
			name:  "error/db_error",
			id:    userID,
			roles: []string{"user"},
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta(
					`UPDATE "auth_users" SET "roles"=$1,"updated_at"=$2 WHERE id = $3 AND "auth_users"."deleted_at" IS NULL`,
				)).WithArgs(
					`["user"]`,
					sqlmock.AnyArg(),
					userID,
				).WillReturnError(errors.New("lock timeout"))
				mock.ExpectRollback()
			},
			wantErr: true,
			errMsg:  "lock timeout",
		},
	}
	
	for _, tc := range tests {
		tc := tc // capture range variable for parallel tests
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			
			db, mock := setupTestDB(t)
			store := NewUserStore(db)
			
			tc.setup(mock)
			
			err := store.UpdateRoles(ctx, tc.id, tc.roles)
			
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

func TestUserStore_BumpSessionVersion(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	userID := uuid.New()
	
	tests := []struct {
		name    string
		id      uuid.UUID
		setup   func(sqlmock.Sqlmock)
		wantErr bool
		errMsg  string
	}{
		{
			name: "ok/bump_version",
			id:   userID,
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta(
					`UPDATE "auth_users" SET "session_version"=session_version + $1,"updated_at"=$2 WHERE id = $3 AND "auth_users"."deleted_at" IS NULL`,
				)).WithArgs(
					1,
					sqlmock.AnyArg(),
					userID,
				).WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit()
			},
			wantErr: false,
		},
		{
			name: "error/user_not_found",
			id:   uuid.New(),
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta(
					`UPDATE "auth_users" SET "session_version"=session_version + $1,"updated_at"=$2 WHERE id = $3 AND "auth_users"."deleted_at" IS NULL`,
				)).WithArgs(
					1,
					sqlmock.AnyArg(),
					sqlmock.AnyArg(),
				).WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectCommit()
			},
			wantErr: true,
			errMsg:  "user not found",
		},
		{
			name: "error/db_error",
			id:   userID,
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta(
					`UPDATE "auth_users" SET "session_version"=session_version + $1,"updated_at"=$2 WHERE id = $3 AND "auth_users"."deleted_at" IS NULL`,
				)).WithArgs(
					1,
					sqlmock.AnyArg(),
					userID,
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
			store := NewUserStore(db)
			
			tc.setup(mock)
			
			err := store.BumpSessionVersion(ctx, tc.id)
			
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

func TestUserStore_toDomain(t *testing.T) {
	t.Parallel()
	
	userID := uuid.New()
	now := time.Now()
	
	tests := []struct {
		name    string
		user    *User
		want    *domain.User
		wantErr bool
		errMsg  string
	}{
		{
			name: "ok/basic_user",
			user: &User{
				ID:             userID,
				Email:          "user@example.com",
				RolesJSON:      `["user"]`,
				SessionVersion: 1,
				CreatedAt:      now,
				UpdatedAt:      now,
			},
			want: &domain.User{
				ID:             userID,
				Email:          "user@example.com",
				Roles:          []string{"user"},
				SessionVersion: 1,
				CreatedAt:      now,
				UpdatedAt:      now,
			},
			wantErr: false,
		},
		{
			name: "ok/multiple_roles",
			user: &User{
				ID:             userID,
				Email:          "admin@example.com",
				RolesJSON:      `["user","admin","editor"]`,
				SessionVersion: 5,
				CreatedAt:      now,
				UpdatedAt:      now,
			},
			want: &domain.User{
				ID:             userID,
				Email:          "admin@example.com",
				Roles:          []string{"user", "admin", "editor"},
				SessionVersion: 5,
				CreatedAt:      now,
				UpdatedAt:      now,
			},
			wantErr: false,
		},
		{
			name: "ok/empty_roles",
			user: &User{
				ID:             userID,
				Email:          "noroles@example.com",
				RolesJSON:      `[]`,
				SessionVersion: 1,
				CreatedAt:      now,
				UpdatedAt:      now,
			},
			want: &domain.User{
				ID:             userID,
				Email:          "noroles@example.com",
				Roles:          []string{},
				SessionVersion: 1,
				CreatedAt:      now,
				UpdatedAt:      now,
			},
			wantErr: false,
		},
		{
			name: "error/invalid_json",
			user: &User{
				ID:             userID,
				Email:          "badjson@example.com",
				RolesJSON:      `{invalid json`,
				SessionVersion: 1,
				CreatedAt:      now,
				UpdatedAt:      now,
			},
			wantErr: true,
			errMsg:  "invalid character",
		},
		{
			name: "error/not_array",
			user: &User{
				ID:             userID,
				Email:          "notarray@example.com",
				RolesJSON:      `{"role": "user"}`,
				SessionVersion: 1,
				CreatedAt:      now,
				UpdatedAt:      now,
			},
			wantErr: true,
			errMsg:  "cannot unmarshal",
		},
	}
	
	for _, tc := range tests {
		tc := tc // capture range variable for parallel tests
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			
			db, _ := setupTestDB(t)
			store := NewUserStore(db)
			
			got, err := store.toDomain(tc.user)
			
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
				assert.Equal(t, tc.want.SessionVersion, got.SessionVersion)
				assert.Equal(t, tc.want.CreatedAt.Unix(), got.CreatedAt.Unix())
				assert.Equal(t, tc.want.UpdatedAt.Unix(), got.UpdatedAt.Unix())
			}
		})
	}
}

func TestUserStore_RoleJSONMarshaling(t *testing.T) {
	t.Parallel()
	
	tests := []struct {
		name    string
		roles   []string
		want    string
		wantErr bool
	}{
		{
			name:    "ok/single_role",
			roles:   []string{"user"},
			want:    `["user"]`,
			wantErr: false,
		},
		{
			name:    "ok/multiple_roles",
			roles:   []string{"user", "admin", "editor"},
			want:    `["user","admin","editor"]`,
			wantErr: false,
		},
		{
			name:    "ok/empty_roles",
			roles:   []string{},
			want:    `[]`,
			wantErr: false,
		},
		{
			name:    "ok/nil_roles",
			roles:   nil,
			want:    `null`,
			wantErr: false,
		},
		{
			name:    "ok/special_chars",
			roles:   []string{"user-admin", "super_user", "role@special"},
			want:    `["user-admin","super_user","role@special"]`,
			wantErr: false,
		},
	}
	
	for _, tc := range tests {
		tc := tc // capture range variable for parallel tests
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			
			got, err := json.Marshal(tc.roles)
			
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.want, string(got))
				
				// Test unmarshaling back
				var roles []string
				err = json.Unmarshal(got, &roles)
				require.NoError(t, err)
				if tc.roles == nil {
					assert.Nil(t, roles)
				} else {
					assert.Equal(t, tc.roles, roles)
				}
			}
		})
	}
}

func TestUserStore_ConcurrentOperations(t *testing.T) {
	// Note: This test would require a real database connection
	// to properly test concurrent operations. With sqlmock,
	// we can only verify the SQL expectations are met.
	t.Skip("Concurrent operations require real database")
}

func TestUserStore_TransactionRollback(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	
	t.Run("rollback_on_error", func(t *testing.T) {
		t.Parallel()
		
		db, mock := setupTestDB(t)
		store := NewUserStore(db)
		
		// Expect transaction to start, fail, and rollback
		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(
			`INSERT INTO "auth_users" ("id","email","roles","session_version","created_at","updated_at","deleted_at") VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		)).WithArgs(
			sqlmock.AnyArg(),
			"fail@example.com",
			`["user"]`,
			int64(1),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			nil,
		).WillReturnError(sql.ErrConnDone)
		mock.ExpectRollback()
		
		_, err := store.Create(ctx, "fail@example.com", []string{"user"})
		require.Error(t, err)
		
		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})
}