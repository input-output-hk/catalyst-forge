package user

import (
	"testing"
	"time"

	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/user"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestBootstrapTokenRepository(t *testing.T) {
	// Setup in-memory database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// Run migrations
	err = db.AutoMigrate(&user.BootstrapToken{})
	require.NoError(t, err)

	repo := NewBootstrapTokenRepository(db)

	t.Run("Create and retrieve bootstrap token", func(t *testing.T) {
		token := &user.BootstrapToken{
			TokenHash:   "test_token_hash_123",
			UsedAt:      time.Now(),
			UsedByEmail: "admin@example.com",
			InviteID:    1,
		}

		// Create token
		err := repo.Create(token)
		assert.NoError(t, err)
		assert.NotZero(t, token.ID)

		// Retrieve by hash
		retrieved, err := repo.GetByTokenHash("test_token_hash_123")
		assert.NoError(t, err)
		assert.NotNil(t, retrieved)
		assert.Equal(t, token.TokenHash, retrieved.TokenHash)
		assert.Equal(t, token.UsedByEmail, retrieved.UsedByEmail)
	})

	t.Run("Check if token is used", func(t *testing.T) {
		// Check non-existent token
		used, err := repo.IsTokenUsed("non_existent_hash")
		assert.NoError(t, err)
		assert.False(t, used)

		// Create a token
		token := &user.BootstrapToken{
			TokenHash:   "used_token_hash",
			UsedAt:      time.Now(),
			UsedByEmail: "user@example.com",
			InviteID:    2,
		}
		err = repo.Create(token)
		require.NoError(t, err)

		// Check if it's used
		used, err = repo.IsTokenUsed("used_token_hash")
		assert.NoError(t, err)
		assert.True(t, used)
	})

	t.Run("GetByTokenHash returns nil for non-existent token", func(t *testing.T) {
		token, err := repo.GetByTokenHash("does_not_exist")
		assert.NoError(t, err)
		assert.Nil(t, token)
	})

	t.Run("Create returns error for nil token", func(t *testing.T) {
		err := repo.Create(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be nil")
	})
}
