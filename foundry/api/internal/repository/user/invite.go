package user

import (
	"time"

	dbmodel "github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/user"
	"gorm.io/gorm"
)

type InviteRepository interface {
	Create(inv *dbmodel.Invite) error
	GetByID(id uint) (*dbmodel.Invite, error)
	GetByTokenHash(hash string) (*dbmodel.Invite, error)
	MarkRedeemed(id uint) error
	IncrementFailedAttempts(id uint, lockoutDuration time.Duration, maxAttempts int) error
	ResetFailedAttempts(id uint) error
	IsLocked(id uint) (bool, error)
}

type inviteRepository struct {
	db *gorm.DB
}

func NewInviteRepository(db *gorm.DB) InviteRepository { return &inviteRepository{db: db} }

func (r *inviteRepository) Create(inv *dbmodel.Invite) error { return r.db.Create(inv).Error }

func (r *inviteRepository) GetByID(id uint) (*dbmodel.Invite, error) {
	var out dbmodel.Invite
	tx := r.db.First(&out, id)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return &out, nil
}

func (r *inviteRepository) GetByTokenHash(hash string) (*dbmodel.Invite, error) {
	var out dbmodel.Invite
	tx := r.db.First(&out, "token_hash = ?", hash)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return &out, nil
}

func (r *inviteRepository) MarkRedeemed(id uint) error {
	now := time.Now()
	tx := r.db.Model(&dbmodel.Invite{}).Where("id = ?", id).Update("redeemed_at", &now)
	return tx.Error
}

// IncrementFailedAttempts increments the failed attempt counter and potentially locks the invite.
func (r *inviteRepository) IncrementFailedAttempts(id uint, lockoutDuration time.Duration, maxAttempts int) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var invite dbmodel.Invite
		if err := tx.First(&invite, id).Error; err != nil {
			return err
		}

		now := time.Now()
		invite.FailedAttempts++
		invite.LastAttemptAt = &now

		// Lock the invite if max attempts exceeded
		if invite.FailedAttempts >= maxAttempts {
			lockUntil := now.Add(lockoutDuration)
			invite.LockedUntil = &lockUntil
		}

		return tx.Save(&invite).Error
	})
}

// ResetFailedAttempts resets the failed attempt counter (called on successful verification).
func (r *inviteRepository) ResetFailedAttempts(id uint) error {
	return r.db.Model(&dbmodel.Invite{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"failed_attempts": 0,
			"last_attempt_at": nil,
			"locked_until":    nil,
		}).Error
}

// IsLocked checks if an invite is currently locked due to too many failed attempts.
func (r *inviteRepository) IsLocked(id uint) (bool, error) {
	var invite dbmodel.Invite
	if err := r.db.First(&invite, id).Error; err != nil {
		return false, err
	}

	if invite.LockedUntil == nil {
		return false, nil
	}

	// Check if lockout period has expired
	if time.Now().After(*invite.LockedUntil) {
		// Lockout expired, reset the lock
		_ = r.ResetFailedAttempts(id)
		return false, nil
	}

	return true, nil
}
