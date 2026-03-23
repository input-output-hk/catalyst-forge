package user

import (
	"testing"
	"time"

	dbuser "github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/user"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestRefreshTokenRepository_chainOps(t *testing.T) {
	tests := []struct {
		name     string
		validate func(t *testing.T, repo RefreshTokenRepository, db *gorm.DB)
	}{
		{
			name: "create_get_mark_replaced_revoke_touch",
			validate: func(t *testing.T, repo RefreshTokenRepository, db *gorm.DB) {
				// create two tokens in a chain
				t1 := &dbuser.RefreshToken{UserID: 1, TokenHash: "h1", ExpiresAt: time.Now().Add(24 * time.Hour)}
				require.NoError(t, repo.Create(t1))
				t2 := &dbuser.RefreshToken{UserID: 1, TokenHash: "h2", ExpiresAt: time.Now().Add(24 * time.Hour)}
				require.NoError(t, repo.Create(t2))

				// get by hash
				got, err := repo.GetByHash("h1")
				require.NoError(t, err)
				assert.Equal(t, t1.ID, got.ID)

				// mark replaced
				require.NoError(t, repo.MarkReplaced(t1.ID, t2.ID))

				// touch usage
				require.NoError(t, repo.TouchUsage(t2.ID, time.Now()))

				// revoke chain starting at first token revokes only that id and any with replaced_by = startID
				require.NoError(t, repo.RevokeChain(t1.ID))
				var out []dbuser.RefreshToken
				require.NoError(t, db.Find(&out).Error)
				var revoked, notRevoked int
				for _, rt := range out {
					if rt.RevokedAt != nil {
						revoked++
					} else {
						notRevoked++
					}
				}
				assert.Equal(t, 1, revoked)
				assert.Equal(t, 1, notRevoked)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
			require.NoError(t, err)
			require.NoError(t, db.AutoMigrate(&dbuser.RefreshToken{}))
			repo := NewRefreshTokenRepository(db)
			tt.validate(t, repo, db)
		})
	}
}
