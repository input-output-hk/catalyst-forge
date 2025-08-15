package gormstore

import (
	"context"
	"errors"
	"time"

	repodb "github.com/catalystgo/catalyst-forge/lib/foundry/db"
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

// dbFor returns the appropriate database handle for the given context.
func (s *RecoveryCodeStore) dbFor(ctx context.Context) *gorm.DB {
	if tx := repodb.TxFromContext(ctx); tx != nil {
		return tx
	}
	return s.db
}

// ReplaceCodes replaces all recovery codes for a user with new ones.
//
// This operation is atomic - either all codes are replaced or none are.
func (s *RecoveryCodeStore) ReplaceCodes(ctx context.Context, userID uuid.UUID, codeHashes [][]byte) error {
	// Start a transaction if not already in one
	db := s.dbFor(ctx)
	var tx *gorm.DB
	if db == s.db {
		// We're not in a transaction, start one
		tx = db.WithContext(ctx).Begin()
	} else {
		// Already in a transaction, use it
		tx = db
	}
	defer func() {
		if r := recover(); r != nil {
			// Only rollback if we created the transaction
			if tx != s.dbFor(ctx) {
				tx.Rollback()
			}
		}
	}()

	// Delete all existing codes for the user
	if err := tx.Where("user_id = ?", userID).Delete(&RecoveryCode{}).Error; err != nil {
		// Only rollback if we created the transaction
		if tx != s.dbFor(ctx) {
			tx.Rollback()
		}
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

	// Commit transaction only if we created it
	if tx != s.dbFor(ctx) {
		return tx.Commit().Error
	}
	return nil
}

// Consume attempts to use a recovery code, marking it as used if found.
//
// Returns true if the code was valid and consumed, false otherwise.
func (s *RecoveryCodeStore) Consume(ctx context.Context, userID uuid.UUID, codeHash []byte, at time.Time) (bool, error) {
	// Start a transaction if not already in one
	db := s.dbFor(ctx)
	var tx *gorm.DB
	if db == s.db {
		// We're not in a transaction, start one
		tx = db.WithContext(ctx).Begin()
	} else {
		// Already in a transaction, use it
		tx = db
	}
	defer func() {
		if r := recover(); r != nil {
			// Only rollback if we created the transaction
			if tx != s.dbFor(ctx) {
				tx.Rollback()
			}
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
		// Only rollback if we created the transaction
		if tx != s.dbFor(ctx) {
			tx.Rollback()
		}
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
	
	if err := s.dbFor(ctx).WithContext(ctx).
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

// CountUnused returns the number of unused recovery codes for a user.
//
// This is used to show users how many codes they have left.
func (s *RecoveryCodeStore) CountUnused(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int64
	
	if err := s.dbFor(ctx).WithContext(ctx).
		Model(&RecoveryCode{}).
		Where("user_id = ? AND used_at IS NULL", userID).
		Count(&count).Error; err != nil {
		return 0, err
	}
	
	return int(count), nil
}