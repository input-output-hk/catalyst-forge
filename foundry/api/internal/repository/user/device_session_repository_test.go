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

func TestDeviceSessionRepository_updates(t *testing.T) {
	tests := []struct {
		name     string
		validate func(t *testing.T, repo DeviceSessionRepository, db *gorm.DB)
	}{
		{
			name: "update_interval_and_increment_poll",
			validate: func(t *testing.T, repo DeviceSessionRepository, db *gorm.DB) {
				sess := &dbuser.DeviceSession{
					DeviceCode:      "dev1",
					UserCode:        "USER-1",
					ExpiresAt:       time.Now().Add(10 * time.Minute),
					IntervalSeconds: 5,
					Status:          "pending",
				}
				require.NoError(t, repo.Create(sess))
				require.NotZero(t, sess.ID)

				// change interval
				require.NoError(t, repo.UpdateInterval(sess.ID, 9))
				fetched, err := repo.GetByDeviceCode("dev1")
				require.NoError(t, err)
				assert.Equal(t, 9, fetched.IntervalSeconds)

				// increment poll
				require.NoError(t, repo.IncrementPollCount(sess.ID))
				fetched2, err := repo.GetByDeviceCode("dev1")
				require.NoError(t, err)
				assert.Equal(t, 1, fetched2.PollCount)
			},
		},
		{
			name: "approve_by_user_code_sets_fields",
			validate: func(t *testing.T, repo DeviceSessionRepository, db *gorm.DB) {
				sess := &dbuser.DeviceSession{
					DeviceCode:      "dev2",
					UserCode:        "ABCD-1234",
					ExpiresAt:       time.Now().Add(10 * time.Minute),
					IntervalSeconds: 5,
					Status:          "pending",
				}
				require.NoError(t, repo.Create(sess))

				require.NoError(t, repo.ApproveByUserCode("ABCD-1234", 42))
				fetched, err := repo.GetByUserCode("ABCD-1234")
				require.NoError(t, err)
				assert.Equal(t, "approved", fetched.Status)
				require.NotNil(t, fetched.ApprovedUserID)
				assert.Equal(t, uint(42), *fetched.ApprovedUserID)
				// completed_at is not set by ApproveByUserCode in current implementation
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
			require.NoError(t, err)
			require.NoError(t, db.AutoMigrate(&dbuser.DeviceSession{}))
			repo := NewDeviceSessionRepository(db)
			tt.validate(t, repo, db)
		})
	}
}
