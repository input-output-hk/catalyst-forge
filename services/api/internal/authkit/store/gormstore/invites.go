package gormstore

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/domain"
	repodb "github.com/catalystgo/catalyst-forge/lib/foundry/db"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// InviteStore is a GORM-based implementation of store.InviteStore.
type InviteStore struct {
	db *gorm.DB
}

// NewInviteStore creates a new GORM-based invite store.
func NewInviteStore(db *gorm.DB) *InviteStore {
	return &InviteStore{db: db}
}

// dbFor returns the appropriate database handle for the given context.
func (s *InviteStore) dbFor(ctx context.Context) *gorm.DB {
	if tx := repodb.TxFromContext(ctx); tx != nil {
		return tx
	}
	return s.db
}

// Create stores a new invite.
func (s *InviteStore) Create(ctx context.Context, inv *domain.Invite) error {
	rolesJSON, err := json.Marshal(inv.Roles)
	if err != nil {
		return err
	}

	dbInvite := &Invite{
		ID:         inv.ID,
		Email:      inv.Email,
		RolesJSON:  string(rolesJSON),
		TokenHash:  inv.TokenHash,
		ExpiresAt:  inv.ExpiresAt,
		Attempts:   inv.Attempts,
		RedeemedAt: inv.RedeemedAt,
		CreatedBy:  inv.CreatedBy,
	}

	if err := s.dbFor(ctx).WithContext(ctx).Create(dbInvite).Error; err != nil {
		return err
	}

	return nil
}

// Get retrieves an invite by its ID.
func (s *InviteStore) Get(ctx context.Context, id uuid.UUID) (*domain.Invite, error) {
	var dbInvite Invite
	
	if err := s.dbFor(ctx).WithContext(ctx).
		Where("id = ?", id).
		First(&dbInvite).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("invite not found")
		}
		return nil, err
	}

	return s.toDomain(&dbInvite)
}

// IncrementAttempts increments the failed attempt counter for an invite.
func (s *InviteStore) IncrementAttempts(ctx context.Context, id uuid.UUID) error {
	result := s.dbFor(ctx).WithContext(ctx).Model(&Invite{}).
		Where("id = ? AND redeemed_at IS NULL", id).
		UpdateColumn("attempts", gorm.Expr("attempts + ?", 1))

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("invite not found or already redeemed")
	}

	return nil
}

// Redeem marks an invite as successfully redeemed.
func (s *InviteStore) Redeem(ctx context.Context, id uuid.UUID, at time.Time) error {
	result := s.dbFor(ctx).WithContext(ctx).Model(&Invite{}).
		Where("id = ? AND redeemed_at IS NULL", id).
		Update("redeemed_at", at)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("invite not found or already redeemed")
	}

	return nil
}

// toDomain converts a database model to a domain entity.
func (s *InviteStore) toDomain(inv *Invite) (*domain.Invite, error) {
	var roles []string
	if inv.RolesJSON != "" {
		if err := json.Unmarshal([]byte(inv.RolesJSON), &roles); err != nil {
			// Handle invalid JSON by treating as empty array
			roles = []string{}
		}
	}

	return &domain.Invite{
		ID:         inv.ID,
		Email:      inv.Email,
		Roles:      roles,
		TokenHash:  inv.TokenHash,
		ExpiresAt:  inv.ExpiresAt,
		Attempts:   inv.Attempts,
		RedeemedAt: inv.RedeemedAt,
		CreatedBy:  inv.CreatedBy,
	}, nil
}