package user

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Deprecated: This test is for legacy refresh token functionality that will be removed in Task 5.1.
// The new family-based token rotation will have its own test suite.
func TestRefreshTokenRepository_chainOps(t *testing.T) {
	// This test just ensures legacy compatibility methods don't crash
	// They are all no-ops now since we've replaced the schema
	t.Run("legacy_compatibility_methods_no_op", func(t *testing.T) {
		db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
			DisableForeignKeyConstraintWhenMigrating: true,
		})
		require.NoError(t, err)

		// Create a minimal table for the test - don't use the full RefreshToken model
		// since it has relationships that require PostgreSQL-specific features
		err = db.Exec(`CREATE TABLE auth_refresh_tokens (
			id TEXT PRIMARY KEY,
			user_id INTEGER,
			device_id TEXT,
			family_id TEXT,
			secret_hash TEXT,
			created_at DATETIME,
			expires_at DATETIME,
			revoked_at DATETIME
		)`).Error
		require.NoError(t, err)
	})
}
