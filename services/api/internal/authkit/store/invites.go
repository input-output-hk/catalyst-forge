package store

import (
	"context"
	"time"

	"github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/domain"
	"github.com/google/uuid"
)

// InviteStore handles invite persistence.
type InviteStore interface {
	// Create stores a new invite.
	Create(ctx context.Context, inv *domain.Invite) error
	
	// Get retrieves an invite by its ID.
	Get(ctx context.Context, id uuid.UUID) (*domain.Invite, error)
	
	// IncrementAttempts increments the failed attempt counter for an invite.
	IncrementAttempts(ctx context.Context, id uuid.UUID) error
	
	// Redeem marks an invite as successfully redeemed.
	Redeem(ctx context.Context, id uuid.UUID, at time.Time) error
}