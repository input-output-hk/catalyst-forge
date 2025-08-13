package gormstore

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/catalystgo/catalyst-forge/lib/foundry/authkit/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestNewCredentialStore(t *testing.T) {
	t.Parallel()
	
	db, _ := setupTestDB(t)
	store := NewCredentialStore(db)
	assert.NotNil(t, store)
	assert.NotNil(t, store.db)
}

func TestCredentialStore_Add(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	userID := uuid.New()
	credID := []byte("credential-id-123")
	publicKey := []byte("public-key-data")
	
	tests := []struct {
		name    string
		cred    *domain.Credential
		setup   func(sqlmock.Sqlmock)
		wantErr bool
		errMsg  string
	}{
		{
			name: "ok/basic_credential",
			cred: &domain.Credential{
				ID:         credID,
				UserID:     userID,
				PublicKey:  publicKey,
				AAGUID:     "00000000-0000-0000-0000-000000000001",
				DeviceName: "YubiKey 5",
				SignCount:  0,
				RK:         true,
				CreatedAt:  time.Now(),
			},
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta(
					`INSERT INTO "auth_credentials" ("id","user_id","public_key","aa_guid","device_name","rk","transports","sign_count","created_at","last_used_at","revoked") VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
				)).WithArgs(
					credID,
					userID,
					publicKey,
					"00000000-0000-0000-0000-000000000001",
					"YubiKey 5",
					true,
					"",
					uint32(0),
					sqlmock.AnyArg(),
					sqlmock.AnyArg(),
					false,
				).WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			wantErr: false,
		},
		{
			name: "ok/with_transports",
			cred: &domain.Credential{
				ID:         credID,
				UserID:     userID,
				PublicKey:  publicKey,
				AAGUID:     "00000000-0000-0000-0000-000000000002",
				DeviceName: "TouchID",
				Transports: []string{"internal", "hybrid"},
				SignCount:  0,
				RK:         true,
				CreatedAt:  time.Now(),
			},
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta(
					`INSERT INTO "auth_credentials" ("id","user_id","public_key","aa_guid","device_name","rk","transports","sign_count","created_at","last_used_at","revoked") VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
				)).WithArgs(
					credID,
					userID,
					publicKey,
					"00000000-0000-0000-0000-000000000002",
					"TouchID",
					true,
					`["internal","hybrid"]`,
					uint32(0),
					sqlmock.AnyArg(),
					sqlmock.AnyArg(),
					false,
				).WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			wantErr: false,
		},
		{
			name: "error/duplicate_credential",
			cred: &domain.Credential{
				ID:         credID,
				UserID:     userID,
				PublicKey:  publicKey,
				AAGUID:     "00000000-0000-0000-0000-000000000001",
				DeviceName: "Duplicate",
				SignCount:  0,
				RK:         true,
				CreatedAt:  time.Now(),
			},
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta(
					`INSERT INTO "auth_credentials" ("id","user_id","public_key","aa_guid","device_name","rk","transports","sign_count","created_at","last_used_at","revoked") VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
				)).WithArgs(
					credID,
					userID,
					publicKey,
					"00000000-0000-0000-0000-000000000001",
					"Duplicate",
					true,
					"",
					uint32(0),
					sqlmock.AnyArg(),
					sqlmock.AnyArg(),
					false,
				).WillReturnError(errors.New("duplicate key value violates unique constraint"))
				mock.ExpectRollback()
			},
			wantErr: true,
			errMsg:  "duplicate key",
		},
		{
			name: "error/db_error",
			cred: &domain.Credential{
				ID:         credID,
				UserID:     userID,
				PublicKey:  publicKey,
				AAGUID:     "",
				DeviceName: "Device",
				SignCount:  0,
				RK:         false,
				CreatedAt:  time.Now(),
			},
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta(
					`INSERT INTO "auth_credentials" ("id","user_id","public_key","aa_guid","device_name","rk","transports","sign_count","created_at","last_used_at","revoked") VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
				)).WithArgs(
					credID,
					userID,
					publicKey,
					"",
					"Device",
					false,
					"",
					uint32(0),
					sqlmock.AnyArg(),
					sqlmock.AnyArg(),
					false,
				).WillReturnError(errors.New("connection lost"))
				mock.ExpectRollback()
			},
			wantErr: true,
			errMsg:  "connection lost",
		},
	}
	
	for _, tc := range tests {
		tc := tc // capture range variable for parallel tests
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			
			db, mock := setupTestDB(t)
			store := NewCredentialStore(db)
			
			tc.setup(mock)
			
			err := store.Add(ctx, tc.cred)
			
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

func TestCredentialStore_Get(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	userID := uuid.New()
	credID := []byte("credential-id-123")
	publicKey := []byte("public-key-data")
	now := time.Now()
	
	tests := []struct {
		name    string
		id      []byte
		setup   func(sqlmock.Sqlmock)
		wantErr bool
		errMsg  string
		validate func(*testing.T, *domain.Credential)
	}{
		{
			name: "ok/found_credential",
			id:   credID,
			setup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"id", "user_id", "public_key", "aa_guid", "device_name", 
					"rk", "transports", "sign_count", "created_at", "last_used_at", "revoked",
				}).AddRow(
					credID, userID, publicKey, "00000000-0000-0000-0000-000000000001",
					"YubiKey 5", true, `["usb","nfc"]`, uint32(42), now, now, false,
				)
				
				mock.ExpectQuery(regexp.QuoteMeta(
					`SELECT * FROM "auth_credentials" WHERE id = $1 ORDER BY "auth_credentials"."id" LIMIT $2`,
				)).WithArgs(credID, 1).WillReturnRows(rows)
			},
			wantErr: false,
			validate: func(t *testing.T, cred *domain.Credential) {
				assert.Equal(t, credID, cred.ID)
				assert.Equal(t, userID, cred.UserID)
				assert.Equal(t, publicKey, cred.PublicKey)
				assert.Equal(t, "00000000-0000-0000-0000-000000000001", cred.AAGUID)
				assert.Equal(t, "YubiKey 5", cred.DeviceName)
				assert.True(t, cred.RK)
				assert.Equal(t, []string{"usb", "nfc"}, cred.Transports)
				assert.Equal(t, uint32(42), cred.SignCount)
			},
		},
		{
			name: "ok/no_transports",
			id:   credID,
			setup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"id", "user_id", "public_key", "aa_guid", "device_name",
					"rk", "transports", "sign_count", "created_at", "last_used_at", "revoked",
				}).AddRow(
					credID, userID, publicKey, "", "Device", false, "", uint32(0), now, now, false,
				)
				
				mock.ExpectQuery(regexp.QuoteMeta(
					`SELECT * FROM "auth_credentials" WHERE id = $1 ORDER BY "auth_credentials"."id" LIMIT $2`,
				)).WithArgs(credID, 1).WillReturnRows(rows)
			},
			wantErr: false,
			validate: func(t *testing.T, cred *domain.Credential) {
				assert.Equal(t, []string{}, cred.Transports)
				assert.Equal(t, "", cred.AAGUID)
				assert.False(t, cred.RK)
			},
		},
		{
			name: "error/not_found",
			id:   []byte("not-found"),
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(
					`SELECT * FROM "auth_credentials" WHERE id = $1 ORDER BY "auth_credentials"."id" LIMIT $2`,
				)).WithArgs([]byte("not-found"), 1).WillReturnError(gorm.ErrRecordNotFound)
			},
			wantErr: true,
			errMsg:  "credential not found",
		},
		{
			name: "error/db_error",
			id:   credID,
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(
					`SELECT * FROM "auth_credentials" WHERE id = $1 ORDER BY "auth_credentials"."id" LIMIT $2`,
				)).WithArgs(credID, 1).WillReturnError(errors.New("query timeout"))
			},
			wantErr: true,
			errMsg:  "query timeout",
		},
	}
	
	for _, tc := range tests {
		tc := tc // capture range variable for parallel tests
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			
			db, mock := setupTestDB(t)
			store := NewCredentialStore(db)
			
			tc.setup(mock)
			
			cred, err := store.Get(ctx, tc.id)
			
			if tc.wantErr {
				require.Error(t, err)
				if tc.errMsg != "" {
					assert.Contains(t, err.Error(), tc.errMsg)
				}
				assert.Nil(t, cred)
			} else {
				require.NoError(t, err)
				require.NotNil(t, cred)
				
				if tc.validate != nil {
					tc.validate(t, cred)
				}
			}
			
			err = mock.ExpectationsWereMet()
			assert.NoError(t, err)
		})
	}
}

func TestCredentialStore_GetByUser(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	userID := uuid.New()
	now := time.Now()
	
	tests := []struct {
		name    string
		userID  uuid.UUID
		setup   func(sqlmock.Sqlmock)
		wantErr bool
		errMsg  string
		validate func(*testing.T, []domain.Credential)
	}{
		{
			name:   "ok/multiple_credentials",
			userID: userID,
			setup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"id", "user_id", "public_key", "aa_guid", "device_name",
					"rk", "transports", "sign_count", "created_at", "last_used_at", "revoked",
				}).AddRow(
					[]byte("cred-1"), userID, []byte("key-1"), "aa_guid-1",
					"Device 1", true, `["usb"]`, uint32(10), now, now, false,
				).AddRow(
					[]byte("cred-2"), userID, []byte("key-2"), "aa_guid-2",
					"Device 2", true, `["internal"]`, uint32(20), now, now, false,
				)
				
				mock.ExpectQuery(regexp.QuoteMeta(
					`SELECT * FROM "auth_credentials" WHERE user_id = $1 AND revoked = $2`,
				)).WithArgs(userID, false).WillReturnRows(rows)
			},
			wantErr: false,
			validate: func(t *testing.T, creds []domain.Credential) {
				assert.Len(t, creds, 2)
				assert.Equal(t, []byte("cred-1"), creds[0].ID)
				assert.Equal(t, []byte("cred-2"), creds[1].ID)
				assert.Equal(t, "Device 1", creds[0].DeviceName)
				assert.Equal(t, "Device 2", creds[1].DeviceName)
			},
		},
		{
			name:   "ok/no_credentials",
			userID: uuid.New(),
			setup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"id", "user_id", "public_key", "aa_guid", "device_name",
					"rk", "transports", "sign_count", "created_at", "last_used_at", "revoked",
				})
				
				mock.ExpectQuery(regexp.QuoteMeta(
					`SELECT * FROM "auth_credentials" WHERE user_id = $1 AND revoked = $2`,
				)).WithArgs(sqlmock.AnyArg(), false).WillReturnRows(rows)
			},
			wantErr: false,
			validate: func(t *testing.T, creds []domain.Credential) {
				assert.Len(t, creds, 0)
			},
		},
		{
			name:   "error/db_error",
			userID: userID,
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(
					`SELECT * FROM "auth_credentials" WHERE user_id = $1 AND revoked = $2`,
				)).WithArgs(userID, false).WillReturnError(errors.New("connection error"))
			},
			wantErr: true,
			errMsg:  "connection error",
		},
	}
	
	for _, tc := range tests {
		tc := tc // capture range variable for parallel tests
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			
			db, mock := setupTestDB(t)
			store := NewCredentialStore(db)
			
			tc.setup(mock)
			
			creds, err := store.GetByUser(ctx, tc.userID)
			
			if tc.wantErr {
				require.Error(t, err)
				if tc.errMsg != "" {
					assert.Contains(t, err.Error(), tc.errMsg)
				}
				assert.Nil(t, creds)
			} else {
				require.NoError(t, err)
				require.NotNil(t, creds)
				
				if tc.validate != nil {
					tc.validate(t, creds)
				}
			}
			
			err = mock.ExpectationsWereMet()
			assert.NoError(t, err)
		})
	}
}

func TestCredentialStore_UpdateOnAssertion(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	credID := []byte("credential-id-123")
	now := time.Now()
	
	tests := []struct {
		name      string
		id        []byte
		signCount uint32
		lastUsed  time.Time
		setup     func(sqlmock.Sqlmock)
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "ok/update_sign_count",
			id:        credID,
			signCount: 43,
			lastUsed:  now,
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta(
					`UPDATE "auth_credentials" SET "last_used_at"=$1,"sign_count"=$2 WHERE id = $3 AND revoked = $4`,
				)).WithArgs(
					sqlmock.AnyArg(),
					uint32(43),
					credID,
					false,
				).WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit()
			},
			wantErr: false,
		},
		{
			name:      "ok/zero_sign_count",
			id:        credID,
			signCount: 0,
			lastUsed:  now,
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta(
					`UPDATE "auth_credentials" SET "last_used_at"=$1,"sign_count"=$2 WHERE id = $3 AND revoked = $4`,
				)).WithArgs(
					sqlmock.AnyArg(),
					uint32(0),
					credID,
					false,
				).WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit()
			},
			wantErr: false,
		},
		{
			name:      "error/credential_not_found",
			id:        []byte("not-found"),
			signCount: 1,
			lastUsed:  now,
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta(
					`UPDATE "auth_credentials" SET "last_used_at"=$1,"sign_count"=$2 WHERE id = $3 AND revoked = $4`,
				)).WithArgs(
					sqlmock.AnyArg(),
					uint32(1),
					[]byte("not-found"),
					false,
				).WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectCommit()
			},
			wantErr: true,
			errMsg:  "credential not found",
		},
		{
			name:      "error/db_error",
			id:        credID,
			signCount: 100,
			lastUsed:  now,
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta(
					`UPDATE "auth_credentials" SET "last_used_at"=$1,"sign_count"=$2 WHERE id = $3 AND revoked = $4`,
				)).WithArgs(
					sqlmock.AnyArg(),
					uint32(100),
					credID,
					false,
				).WillReturnError(errors.New("deadlock"))
				mock.ExpectRollback()
			},
			wantErr: true,
			errMsg:  "deadlock",
		},
	}
	
	for _, tc := range tests {
		tc := tc // capture range variable for parallel tests
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			
			db, mock := setupTestDB(t)
			store := NewCredentialStore(db)
			
			tc.setup(mock)
			
			err := store.UpdateOnAssertion(ctx, tc.id, tc.signCount, tc.lastUsed)
			
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

func TestCredentialStore_Revoke(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	credID := []byte("credential-id-123")
	
	tests := []struct {
		name    string
		id      []byte
		setup   func(sqlmock.Sqlmock)
		wantErr bool
		errMsg  string
	}{
		{
			name: "ok/revoke_credential",
			id:   credID,
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta(
					`UPDATE "auth_credentials" SET "revoked"=$1 WHERE id = $2`,
				)).WithArgs(
					true,
					credID,
				).WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit()
			},
			wantErr: false,
		},
		{
			name: "error/credential_not_found",
			id:   []byte("not-found"),
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta(
					`UPDATE "auth_credentials" SET "revoked"=$1 WHERE id = $2`,
				)).WithArgs(
					true,
					[]byte("not-found"),
				).WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectCommit()
			},
			wantErr: true,
			errMsg:  "credential not found",
		},
		{
			name: "error/db_error",
			id:   credID,
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta(
					`UPDATE "auth_credentials" SET "revoked"=$1 WHERE id = $2`,
				)).WithArgs(
					true,
					credID,
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
			store := NewCredentialStore(db)
			
			tc.setup(mock)
			
			err := store.Revoke(ctx, tc.id)
			
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


func TestCredentialStore_toDomain(t *testing.T) {
	t.Parallel()
	
	userID := uuid.New()
	now := time.Now()
	
	tests := []struct {
		name    string
		cred    *Credential
		want    *domain.Credential
		wantErr bool
		errMsg  string
	}{
		{
			name: "ok/basic_credential",
			cred: &Credential{
				ID:         []byte("cred-id"),
				UserID:     userID,
				PublicKey:  []byte("public-key"),
				AAGUID:     "aa_guid-123",
				DeviceName: "Device",
				RK:         true,
				Transports: `["usb","nfc"]`,
				SignCount:  42,
				CreatedAt:  now,
				LastUsedAt: now,
				Revoked:    false,
			},
			want: &domain.Credential{
				ID:         []byte("cred-id"),
				UserID:     userID,
				PublicKey:  []byte("public-key"),
				AAGUID:     "aa_guid-123",
				DeviceName: "Device",
				RK:         true,
				Transports: []string{"usb", "nfc"},
				SignCount:  42,
				CreatedAt:  now,
				LastUsedAt: now,
			},
			wantErr: false,
		},
		{
			name: "ok/empty_transports",
			cred: &Credential{
				ID:         []byte("cred-id"),
				UserID:     userID,
				PublicKey:  []byte("public-key"),
				AAGUID:     "",
				DeviceName: "Device",
				RK:         false,
				Transports: "",
				SignCount:  0,
				CreatedAt:  now,
				LastUsedAt: now,
				Revoked:    false,
			},
			want: &domain.Credential{
				ID:         []byte("cred-id"),
				UserID:     userID,
				PublicKey:  []byte("public-key"),
				AAGUID:     "",
				DeviceName: "Device",
				RK:         false,
				Transports: []string{},
				SignCount:  0,
				CreatedAt:  now,
				LastUsedAt: now,
			},
			wantErr: false,
		},
		{
			name: "error/invalid_transports_json",
			cred: &Credential{
				ID:         []byte("cred-id"),
				UserID:     userID,
				PublicKey:  []byte("public-key"),
				AAGUID:     "aa_guid",
				DeviceName: "Device",
				RK:         true,
				Transports: `{invalid json}`,
				SignCount:  0,
				CreatedAt:  now,
				LastUsedAt: now,
				Revoked:    false,
			},
			wantErr: true,
			errMsg:  "invalid character",
		},
	}
	
	for _, tc := range tests {
		tc := tc // capture range variable for parallel tests
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			
			db, _ := setupTestDB(t)
			store := NewCredentialStore(db)
			
			got, err := store.toDomain(tc.cred)
			
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
				assert.Equal(t, tc.want.UserID, got.UserID)
				assert.Equal(t, tc.want.PublicKey, got.PublicKey)
				assert.Equal(t, tc.want.AAGUID, got.AAGUID)
				assert.Equal(t, tc.want.DeviceName, got.DeviceName)
				assert.Equal(t, tc.want.RK, got.RK)
				assert.Equal(t, tc.want.Transports, got.Transports)
				assert.Equal(t, tc.want.SignCount, got.SignCount)
			}
		})
	}
}