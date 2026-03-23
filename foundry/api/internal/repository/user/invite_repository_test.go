package user

import (
	"testing"
	"time"

	dbuser "github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/user"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestInviteRepository_basicOps(t *testing.T) {
	tests := []struct {
		name     string
		validate func(t *testing.T, repo InviteRepository, db *gorm.DB)
	}{
		{
			name: "create_get_mark_redeemed",
			validate: func(t *testing.T, repo InviteRepository, db *gorm.DB) {
				inv := &dbuser.Invite{
					Email:     "user@example.com",
					Roles:     pq.StringArray{"admin", "viewer"},
					TokenHash: "hash-abc",
					ExpiresAt: time.Now().Add(1 * time.Hour),
				}
				require.NoError(t, repo.Create(inv))
				require.NotZero(t, inv.ID)

				got, err := repo.GetByTokenHash("hash-abc")
				require.NoError(t, err)
				assert.Equal(t, inv.ID, got.ID)

				require.NoError(t, repo.MarkRedeemed(inv.ID))
				got2, err := repo.GetByID(inv.ID)
				require.NoError(t, err)
				require.NotNil(t, got2.RedeemedAt)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
			require.NoError(t, err)
			require.NoError(t, db.AutoMigrate(&dbuser.Invite{}))
			repo := NewInviteRepository(db)
			tt.validate(t, repo, db)
		})
	}
}
