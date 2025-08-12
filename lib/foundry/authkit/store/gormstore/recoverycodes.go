package gormstore

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// RecoveryCodeStore is a GORM-based implementation of store.RecoveryCodeStore.
type RecoveryCodeStore struct {
	db *gorm.DB
}

// NewRecoveryCodeStore creates a new GORM-based recovery code store.
func NewRecoveryCodeStore(db *gorm.DB) *RecoveryCodeStore {
	return &RecoveryCodeStore{db: db}
}

// ReplaceCodes replaces all recovery codes for a user with new ones.
//
// This operation is atomic - either all codes are replaced or none are.
func (s *RecoveryCodeStore) ReplaceCodes(ctx context.Context, userID uuid.UUID, codeHashes [][]byte) error {
	// Start a transaction
	tx := s.db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Delete all existing codes for the user
	if err := tx.Where("user_id = ?", userID).Delete(&RecoveryCode{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Create new codes
	now := time.Now()
	for _, hash := range codeHashes {
		code := &RecoveryCode{
			ID:        uuid.New(),
			UserID:    userID,
			Hash:      hash,
			CreatedAt: now,
		}
		if err := tx.Create(code).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	// Commit transaction
	return tx.Commit().Error
}

// Consume attempts to use a recovery code, marking it as used if found.
//
// Returns true if the code was valid and consumed, false otherwise.
func (s *RecoveryCodeStore) Consume(ctx context.Context, userID uuid.UUID, codeHash []byte, at time.Time) (bool, error) {
	// Start a transaction
	tx := s.db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Find the code with a lock
	var code RecoveryCode
	err := tx.Set("gorm:query_option", "FOR UPDATE").
		Where("user_id = ? AND hash = ? AND used_at IS NULL", userID, codeHash).
		First(&code).Error

	if err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}

	// Mark as used
	code.UsedAt = &at
	if err := tx.Save(&code).Error; err != nil {
		tx.Rollback()
		return false, err
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return false, err
	}

	return true, nil
}

// List returns all recovery code hashes for a user.
//
// This is primarily for admin operations.
func (s *RecoveryCodeStore) List(ctx context.Context, userID uuid.UUID) ([][]byte, error) {
	var codes []RecoveryCode
	
	if err := s.db.WithContext(ctx).
		Where("user_id = ? AND used_at IS NULL", userID).
		Find(&codes).Error; err != nil {
		return nil, err
	}

	hashes := make([][]byte, 0, len(codes))
	for _, code := range codes {
		// Return a copy of the hash
		hashCopy := make([]byte, len(code.Hash))
		copy(hashCopy, code.Hash)
		hashes = append(hashes, hashCopy)
	}

	return hashes, nil
}